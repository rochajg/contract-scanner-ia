package pdfpipeline

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"contract-scanner/internal/usecase/providers"
)

// PipelineExtractor wraps Pipeline and implements providers.PDFExtractor so it
// can be used as a drop-in replacement for the existing HybridExtractor.
type PipelineExtractor struct {
	pipeline *Pipeline
}

// NewPipelineExtractor creates a PipelineExtractor configured from environment
// variables (same keys as the legacy HybridExtractor).
func NewPipelineExtractor() *PipelineExtractor {
	return &PipelineExtractor{pipeline: NewPipeline(ConfigFromEnv())}
}

// NewPipelineExtractorWithConfig creates a PipelineExtractor with explicit config.
func NewPipelineExtractorWithConfig(config Config) *PipelineExtractor {
	return &PipelineExtractor{pipeline: NewPipeline(config)}
}

// Extract implements providers.PDFExtractor.
// It runs the full pipeline and maps the result to the providers.ExtractResult
// that the rest of the system already understands.
func (e *PipelineExtractor) Extract(ctx context.Context, filePath string) (*providers.ExtractResult, error) {
	docID := docIDFromPath(filePath)

	doc, err := e.pipeline.ProcessPDF(ctx, filePath, docID)
	if err != nil {
		return nil, fmt.Errorf("pdf pipeline: %w", err)
	}

	if strings.TrimSpace(doc.AggregatedText) == "" {
		return nil, fmt.Errorf("unable to extract readable contract text")
	}

	source := documentLevelSource(doc.Metadata.Stats)
	quality := scoreDocumentQuality(doc)

	warnings := make([]string, 0, len(doc.Metadata.Warnings)+1)
	warnings = append(warnings, doc.Metadata.Warnings...)
	if quality < 0.35 {
		warnings = append(warnings, "text extraction quality is low; analysis may be partial")
	}

	pageInfos := make([]providers.PageInfo, len(doc.Pages))
	for i, p := range doc.Pages {
		pageInfos[i] = providers.PageInfo{
			PageNumber: p.PageNumber,
			SourceType: string(p.SourceType),
			CharCount:  len([]rune(p.CleanText)),
		}
	}

	return &providers.ExtractResult{
		Text:             doc.AggregatedText,
		Source:           source,
		QualityScore:     quality,
		Warnings:         dedupeStrings(warnings),
		Pages:            pageInfos,
		HeaderCandidates: doc.Metadata.HeaderCandidates,
		FooterCandidates: doc.Metadata.FooterCandidates,
		TotalPages:       doc.Metadata.Stats.TotalPages,
	}, nil
}

// ConfigFromEnv reads the pipeline configuration from the same environment
// variables used by the legacy HybridExtractor, ensuring backward compatibility.
func ConfigFromEnv() Config {
	cfg := DefaultConfig()

	if v := os.Getenv("PDF_OCR_ENABLED"); v != "" {
		cfg.OCREnabled = strings.EqualFold(v, "true") || v == "1" || strings.EqualFold(v, "yes")
	}
	if v := os.Getenv("PDF_OCR_LANG"); v != "" {
		cfg.OCRLang = v
	}
	if v := os.Getenv("PDF_OCR_DPI"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.OCRDPI = n
		}
	}
	if v := os.Getenv("PDF_MIN_QUALITY_SCORE"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			cfg.MinQualityScore = clampF(f)
		}
	}
	if v := os.Getenv("PDF_OCR_MAX_PAGES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.OCRMaxPages = n
		}
	}
	if v := os.Getenv("PDF_OCR_TIMEOUT_SEC"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.OCRTimeout = time.Duration(n) * time.Second
		}
	}
	return cfg
}

// ── helpers ──────────────────────────────────────────────────────────────────

// docIDFromPath extracts the document identifier from the file path (the
// filename without extension, e.g. "/tmp/abc-123.pdf" → "abc-123").
func docIDFromPath(filePath string) string {
	base := filepath.Base(filePath)
	if idx := strings.LastIndex(base, "."); idx > 0 {
		return base[:idx]
	}
	return base
}

// documentLevelSource maps per-page stats to the "native"/"ocr"/"hybrid"
// string used by the legacy layer.
func documentLevelSource(stats ProcessingStats) string {
	hasNative := stats.NativePages > 0
	hasOCR := stats.OCRPages > 0
	hasHybrid := stats.HybridPages > 0

	if hasOCR || hasHybrid {
		if hasNative {
			return "hybrid"
		}
		return "ocr"
	}
	return "native"
}

// scoreDocumentQuality produces a 0–1 quality score for the extracted document.
// The algorithm mirrors the legacy scoreQuality function so downstream logic
// (enrichAnalysisResult, warnings thresholds) continues to work unchanged.
func scoreDocumentQuality(doc *DocumentExtractionResult) float64 {
	text := strings.TrimSpace(doc.AggregatedText)
	if text == "" {
		return 0
	}

	charLen := len([]rune(text))
	lengthScore := clampF(float64(charLen) / 4000)

	lines := strings.Split(text, "\n")
	lineCount := make(map[string]int, len(lines))
	nonEmpty := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			lineCount[line]++
			nonEmpty++
		}
	}

	uniqueRatio := 0.0
	if nonEmpty > 0 {
		uniqueRatio = float64(len(lineCount)) / float64(nonEmpty)
	}

	dups := 0
	for _, count := range lineCount {
		if count > 1 {
			dups += count - 1
		}
	}
	diversityScore := 1.0
	if nonEmpty > 0 {
		diversityScore = clampF(1 - float64(dups)/float64(nonEmpty))
	}

	legalTerms := []string{
		"contrato", "cláusula", "clausula", "partes", "objeto", "vigência", "vigencia",
		"rescisão", "rescisao", "prazo", "multa", "pagamento", "responsabilidade",
	}
	lower := strings.ToLower(text)
	termsFound := 0
	for _, term := range legalTerms {
		if strings.Contains(lower, term) {
			termsFound++
		}
	}
	termScore := clampF(float64(termsFound) / 6)

	score := 0.40*lengthScore + 0.25*uniqueRatio + 0.20*termScore + 0.15*diversityScore
	return math.Round(clampF(score)*100) / 100
}

func clampF(v float64) float64 {
	if math.IsNaN(v) || v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
