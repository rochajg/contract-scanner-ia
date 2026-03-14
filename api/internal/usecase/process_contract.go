package usecase

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"contract-scanner/internal/usecase/providers"
	"contract-scanner/internal/usecase/repositories"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type IProcessContract interface {
	Execute(ctx context.Context, input ProcessInput) (*ProcessOutput, error)
}

type ProcessInput struct {
	AnalyseID   uuid.UUID
	ClerkUserID string
}

type ProcessOutput struct {
	AnalysisID string          `json:"analysis_id"`
	Status     string          `json:"status"`
	Result     json.RawMessage `json:"result"`
}

type ProcessContract struct {
	AnalyseRepo     repositories.AnalyseRepo
	StorageProvider providers.StorageProvider
	PDFExtractor    providers.PDFExtractor
	LLMProvider     providers.LLMProvider
}

func NewProcessContract(
	analyseRepo repositories.AnalyseRepo,
	storageProvider providers.StorageProvider,
	pdfExtractor providers.PDFExtractor,
	llmProvider providers.LLMProvider,
) IProcessContract {
	return &ProcessContract{
		AnalyseRepo:     analyseRepo,
		StorageProvider: storageProvider,
		PDFExtractor:    pdfExtractor,
		LLMProvider:     llmProvider,
	}
}

func (uc *ProcessContract) Execute(ctx context.Context, input ProcessInput) (*ProcessOutput, error) {
	analyseID := input.AnalyseID.String()

	// 1. Fetch analyse record.
	log.Printf("[process:%s] step=fetch_analyse", analyseID)
	analyse, err := uc.AnalyseRepo.Get(input.AnalyseID)
	if err != nil {
		return nil, fmt.Errorf("analyse not found: %w", err)
	}
	log.Printf("[process:%s] step=fetch_analyse status=%s s3_key=%s extracted_text_key=%v",
		analyseID, analyse.Status, analyse.S3Key, analyse.ExtractedTextS3Key)

	// 2. Ownership check (disabled for local testing).
	// if analyse.ClerkUserID != input.ClerkUserID {
	// 	return nil, fmt.Errorf("forbidden: user does not own this analyse")
	// }

	if analyse.Status == "PROCESSING" {
		return nil, fmt.Errorf("file already processing")
	}

	hasCachedText := analyse.ExtractedTextS3Key != nil && *analyse.ExtractedTextS3Key != ""

	// 3. Verify PDF exists in S3 (skip if we already have cached text and PDF is gone).
	log.Printf("[process:%s] step=head_object s3_key=%s", analyseID, analyse.S3Key)
	if headErr := uc.StorageProvider.HeadObject(ctx, analyse.S3Key); headErr != nil {
		if !hasCachedText {
			return nil, fmt.Errorf("file not uploaded yet: %w", headErr)
		}
		log.Printf("[process:%s] step=head_object PDF not found in S3 but cached text exists; proceeding with retry", analyseID)
	}

	// 4. Mark as PROCESSING.
	analyse.Status = "PROCESSING"
	if err := uc.AnalyseRepo.Update(analyse); err != nil {
		return nil, fmt.Errorf("error updating status: %w", err)
	}

	// 5 & 6. Get contract text — from S3 cache (retry) or by extracting the PDF (first run).
	var text string
	var extractResult *providers.ExtractResult

	if hasCachedText {
		// ── Retry path: load already-extracted text from S3 ──────────────────
		log.Printf("[process:%s] step=load_cached_text key=%s", analyseID, *analyse.ExtractedTextS3Key)
		textBytes, loadErr := uc.StorageProvider.GetObjectBytes(ctx, *analyse.ExtractedTextS3Key)
		if loadErr == nil && len(textBytes) > 0 {
			text = string(textBytes)
			extractResult = &providers.ExtractResult{
				Text:         text,
				Source:       "cached",
				QualityScore: 1.0,
			}
			log.Printf("[process:%s] step=load_cached_text ok chars=%d", analyseID, len(text))
		} else {
			log.Printf("[process:%s] step=load_cached_text failed (%v), falling back to extraction", analyseID, loadErr)
		}
	}

	if extractResult == nil {
		// ── First-run path: download PDF and extract text ─────────────────────
		tmpPath := fmt.Sprintf("/tmp/%s.pdf", analyseID)
		log.Printf("[process:%s] step=download_s3 s3_key=%s dest=%s", analyseID, analyse.S3Key, tmpPath)
		if err := uc.StorageProvider.GetObject(ctx, analyse.S3Key, tmpPath); err != nil {
			return nil, fmt.Errorf("error downloading file: %w", err)
		}

		log.Printf("[process:%s] step=extract_pdf path=%s", analyseID, tmpPath)
		extractResult, err = uc.PDFExtractor.Extract(ctx, tmpPath)
		if err != nil {
			return nil, fmt.Errorf("error extracting text: %w", err)
		}
		text = extractResult.Text
		log.Printf("[process:%s] step=extract_pdf source=%s quality=%.2f chars=%d pages=%d warnings=%v",
			analyseID, extractResult.Source, extractResult.QualityScore, len(text), extractResult.TotalPages, extractResult.Warnings)

		if strings.TrimSpace(text) == "" {
			return nil, fmt.Errorf("unable to extract readable contract text")
		}

		// Persist extracted text to S3 so LLM retries skip PDF processing.
		txtKey := fmt.Sprintf("extracted/%s.txt", analyseID)
		textBytes := []byte(text)
		if putErr := uc.StorageProvider.PutObject(
			ctx, txtKey, bytes.NewReader(textBytes), "text/plain", int64(len(textBytes)),
		); putErr != nil {
			log.Printf("[process:%s] warning: could not save extracted text to S3: %v", analyseID, putErr)
		} else {
			analyse.ExtractedTextS3Key = &txtKey
			if updateErr := uc.AnalyseRepo.Update(analyse); updateErr != nil {
				log.Printf("[process:%s] warning: could not persist extracted_text_s3_key: %v", analyseID, updateErr)
			} else {
				log.Printf("[process:%s] step=save_extracted_text key=%s", analyseID, txtKey)
			}
		}

		// Also write a local debug copy.
		if err := writeDebugTextFile(analyse.ID.String(), analyse.Filename, text); err != nil {
			log.Printf("[process:%s] warning: could not write debug text file: %v", analyseID, err)
		}
	}

	// 7. Call LLM (independent context so HTTP request cancellation doesn't abort it).
	llmCtx, llmCancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer llmCancel()
	log.Printf("[process:%s] step=llm_analyze model=%s", analyseID, analyse.Model)
	resultJSON, err := uc.LLMProvider.AnalyzeContract(llmCtx, text)
	if err != nil {
		log.Printf("[process:%s] step=llm_analyze error=%v", analyseID, err)
		analyse.Status = "FAILED"
		if updateErr := uc.AnalyseRepo.Update(analyse); updateErr != nil {
			log.Printf("[process:%s] warning: could not update status to FAILED: %v", analyseID, updateErr)
		}
		return nil, fmt.Errorf("error analyzing contract: %w", err)
	}
	log.Printf("[process:%s] step=llm_analyze done result_bytes=%d", analyseID, len(resultJSON))

	resultJSON, err = enrichAnalysisResult(resultJSON, extractResult)
	if err != nil {
		return nil, fmt.Errorf("error enriching analysis result: %w", err)
	}

	// 8. Save result and mark COMPLETED.
	analyse.ResultJSON = datatypes.JSON(resultJSON)
	analyse.Status = "COMPLETED"
	now := time.Now().UTC()
	analyse.CompletedAt = &now
	if err := uc.AnalyseRepo.Update(analyse); err != nil {
		return nil, fmt.Errorf("error saving result: %w", err)
	}

	// 9. Return output.
	return &ProcessOutput{
		AnalysisID: analyse.ID.String(),
		Status:     analyse.Status,
		Result:     resultJSON,
	}, nil
}

// writeDebugTextFile saves the extracted text to a local tmp/ folder for inspection.
func writeDebugTextFile(fallbackID string, originalFilename *string, text string) error {
	workDir, err := os.Getwd()
	if err != nil {
		return err
	}
	tmpDir := filepath.Join(workDir, "tmp")
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return err
	}
	filename := buildTempTextFilename(fallbackID, originalFilename)
	return os.WriteFile(filepath.Join(tmpDir, filename), []byte(text), 0o644)
}

func buildTempTextFilename(fallbackID string, originalFilename *string) string {
	if originalFilename == nil || strings.TrimSpace(*originalFilename) == "" {
		return fmt.Sprintf("%s.txt", fallbackID)
	}

	base := filepath.Base(strings.TrimSpace(*originalFilename))
	name := strings.TrimSuffix(base, filepath.Ext(base))
	name = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r
		case r >= 'A' && r <= 'Z':
			return r
		case r >= '0' && r <= '9':
			return r
		case r == '-', r == '_':
			return r
		default:
			return '_'
		}
	}, name)
	name = strings.Trim(name, "_")
	if name == "" {
		name = fallbackID
	}
	return fmt.Sprintf("%s.txt", name)
}

func enrichAnalysisResult(raw json.RawMessage, extractResult *providers.ExtractResult) (json.RawMessage, error) {
	payload := map[string]any{}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("invalid result json: %w", err)
	}

	warnings := []string{}
	if current, ok := payload["analysis_warnings"]; ok {
		switch values := current.(type) {
		case []any:
			for _, value := range values {
				if warning, ok := value.(string); ok {
					trimmed := strings.TrimSpace(warning)
					if trimmed != "" && !slices.Contains(warnings, trimmed) {
						warnings = append(warnings, trimmed)
					}
				}
			}
		case []string:
			for _, warning := range values {
				trimmed := strings.TrimSpace(warning)
				if trimmed != "" && !slices.Contains(warnings, trimmed) {
					warnings = append(warnings, trimmed)
				}
			}
		}
	}

	for _, warning := range extractResult.Warnings {
		trimmed := strings.TrimSpace(warning)
		if trimmed != "" && !slices.Contains(warnings, trimmed) {
			warnings = append(warnings, trimmed)
		}
	}
	if extractResult.QualityScore < 0.35 && !slices.Contains(warnings, "text extraction quality is low; analysis may be partial") {
		warnings = append(warnings, "text extraction quality is low; analysis may be partial")
	}
	if len(warnings) > 0 {
		payload["analysis_warnings"] = warnings
	}

	if _, exists := payload["confidence"]; !exists {
		payload["confidence"] = math.Round(extractResult.QualityScore*100) / 100
	}

	if _, exists := payload["missing_information"]; !exists && extractResult.QualityScore < 0.35 {
		payload["missing_information"] = []string{
			"O texto extraído possui baixa qualidade e pode omitir cláusulas relevantes.",
		}
	}

	out, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(out), nil
}
