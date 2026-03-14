package pdfpipeline

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Config holds all configuration options for the PDF pipeline.
// Environment-variable keys are the same as the legacy HybridExtractor so
// existing deployments need no changes.
type Config struct {
	OCREnabled      bool
	OCRLang         string
	OCRDPI          int
	MinQualityScore float64
	OCRMaxPages     int
	OCRTimeout      time.Duration
}

// DefaultConfig returns sensible production defaults.
func DefaultConfig() Config {
	return Config{
		OCREnabled:      true,
		OCRLang:         "por+eng",
		OCRDPI:          300,
		MinQualityScore: 0.35,
		OCRMaxPages:     80,
		OCRTimeout:      120 * time.Second,
	}
}

// Pipeline orchestrates the full per-page extraction process:
//  1. Native text extraction per page.
//  2. Page classification (native_text / scanned_image / hybrid / unknown).
//  3. OCR for pages that lack a usable text layer.
//  4. Header / footer detection and removal.
//  5. Text cleaning and block segmentation.
//  6. Final aggregation.
type Pipeline struct {
	config          Config
	classifier      PageClassifier
	cleaner         DocumentCleaner
	hfDetector      HeaderFooterDetector
	ocrProvider     OCRProvider
	nativeExtractFn func(filePath string) ([]NativePageText, error) // injectable for tests
}

// NewPipeline creates a fully-wired Pipeline with production components.
func NewPipeline(config Config) *Pipeline {
	p := &Pipeline{
		config:      config,
		classifier:  NewPageClassifier(),
		cleaner:     NewDocumentCleaner(),
		hfDetector:  NewHeaderFooterDetector(),
		ocrProvider: NewTesseractProvider(config.OCRLang),
	}
	p.nativeExtractFn = extractAllPagesNative
	return p
}

// ProcessPDF runs the complete pipeline on a local PDF file and returns a rich
// DocumentExtractionResult. docID is used only for logging and the output struct;
// it does not affect extraction logic.
func (p *Pipeline) ProcessPDF(ctx context.Context, filePath, docID string) (*DocumentExtractionResult, error) {
	start := time.Now()
	log.Printf("[pdfpipeline] start doc_id=%s file=%s", docID, filepath.Base(filePath))

	if _, err := os.Stat(filePath); err != nil {
		return nil, fmt.Errorf("pdf file not accessible: %w", err)
	}

	// ── Step 1: native extraction ──────────────────────────────────────────────
	log.Printf("[pdfpipeline] step=native_extract file=%s", filepath.Base(filePath))
	nativePages, err := p.nativeExtractFn(filePath)
	if err != nil {
		return nil, fmt.Errorf("native extraction: %w", err)
	}
	log.Printf("[pdfpipeline] step=native_extract done total_pages=%d", len(nativePages))

	// ── Step 2: classify pages ─────────────────────────────────────────────────
	pages := make([]PageResult, len(nativePages))
	var needsOCR []int
	var warnings []string

	// Aggregate cleaning stats across pages to avoid per-page noise.
	var pagesWithControlChars []int
	var pagesWithWatermarks []int
	var pagesWithNoise []int

	for i, native := range nativePages {
		rawText := native.Text
		if native.Err != nil {
			log.Printf("[pdfpipeline] page=%d native err: %v", native.PageNumber, native.Err)
			warnings = append(warnings, fmt.Sprintf("page %d: native extraction error: %v", native.PageNumber, native.Err))
			rawText = ""
		}

		sourceType := p.classifier.Classify(rawText)
		cleanText, stats := p.cleaner.Clean(rawText)
		if stats.HadControlChars {
			pagesWithControlChars = append(pagesWithControlChars, native.PageNumber)
		}
		if stats.RemovedWatermark > 0 {
			pagesWithWatermarks = append(pagesWithWatermarks, native.PageNumber)
		}
		if stats.RemovedNoise > 0 {
			pagesWithNoise = append(pagesWithNoise, native.PageNumber)
		}

		log.Printf("[pdfpipeline] page=%d source=%s raw_chars=%d clean_chars=%d",
			native.PageNumber, sourceType, len([]rune(rawText)), len([]rune(cleanText)))

		pages[i] = PageResult{
			PageNumber: native.PageNumber,
			SourceType: sourceType,
			RawText:    rawText,
			CleanText:  cleanText,
			Blocks:     segmentBlocks(cleanText),
		}

		if p.config.OCREnabled &&
			(sourceType == PageSourceScannedImage || sourceType == PageSourceUnknown) {
			needsOCR = append(needsOCR, i)
		}
	}

	// ── Step 3: OCR for scanned/unknown pages ──────────────────────────────────
	ocrUsed := false

	if len(needsOCR) > 0 && p.config.OCREnabled {
		log.Printf("[pdfpipeline] step=ocr pages_needing_ocr=%d/%d", len(needsOCR), len(pages))
		ocrUsed = true

		ocrCtx, ocrCancel := context.WithTimeout(ctx, p.config.OCRTimeout)
		defer ocrCancel()

		tmpDir, tmpErr := os.MkdirTemp("", "pdfpipeline-ocr-*")
		if tmpErr != nil {
			warnings = append(warnings, fmt.Sprintf("could not create OCR temp dir: %v", tmpErr))
		} else {
			defer os.RemoveAll(tmpDir)

			images, rasterErr := rasterizePDF(ocrCtx, filePath, tmpDir, p.config.OCRDPI, p.config.OCRMaxPages)
			if rasterErr != nil {
				log.Printf("[pdfpipeline] rasterize error: %v", rasterErr)
				warnings = append(warnings, fmt.Sprintf("PDF rasterization failed: %v", rasterErr))
			} else {
				imageMap := buildImageMap(images)

				for _, idx := range needsOCR {
					pageNum := pages[idx].PageNumber
					imagePath, ok := imageMap[pageNum]
					if !ok {
						log.Printf("[pdfpipeline] page=%d no rasterized image (beyond OCR limit?)", pageNum)
						continue
					}

					ocrRaw, ocrErr := p.ocrProvider.AnalyzePage(ocrCtx, imagePath)
					if ocrErr != nil {
						log.Printf("[pdfpipeline] page=%d OCR error: %v", pageNum, ocrErr)
						warnings = append(warnings, fmt.Sprintf("page %d: OCR failed: %v", pageNum, ocrErr))
						continue
					}

					cleanOCR, ocrStats := p.cleaner.Clean(ocrRaw)
					if ocrStats.HadControlChars {
						pagesWithControlChars = append(pagesWithControlChars, pageNum)
					}
					if ocrStats.RemovedWatermark > 0 {
						pagesWithWatermarks = append(pagesWithWatermarks, pageNum)
					}
					if ocrStats.RemovedNoise > 0 {
						pagesWithNoise = append(pagesWithNoise, pageNum)
					}

					if strings.TrimSpace(cleanOCR) == "" {
						log.Printf("[pdfpipeline] page=%d OCR produced empty text, keeping native", pageNum)
						continue
					}

					oldSource := pages[idx].SourceType

					if strings.TrimSpace(pages[idx].CleanText) == "" {
						// No native text at all — OCR is the only source.
						pages[idx].RawText = ocrRaw
						pages[idx].CleanText = cleanOCR
						pages[idx].SourceType = PageSourceScannedImage
					} else {
						// Both native and OCR have content — use whichever is richer.
						if len([]rune(cleanOCR)) > len([]rune(pages[idx].CleanText)) {
							pages[idx].RawText = ocrRaw
							pages[idx].CleanText = cleanOCR
						}
						pages[idx].SourceType = PageSourceHybrid
					}
					pages[idx].Blocks = segmentBlocks(pages[idx].CleanText)

					log.Printf("[pdfpipeline] page=%d OCR done source=%s→%s chars=%d",
						pageNum, oldSource, pages[idx].SourceType, len([]rune(pages[idx].CleanText)))
				}
			}
		}
	}

	// ── Step 4: count source types ────────────────────────────────────────────
	stats := ProcessingStats{TotalPages: len(pages), OCRUsed: ocrUsed}
	for _, page := range pages {
		switch page.SourceType {
		case PageSourceNativeText:
			stats.NativePages++
		case PageSourceScannedImage:
			stats.OCRPages++
		case PageSourceHybrid:
			stats.HybridPages++
		default:
			stats.UnknownPages++
		}
	}

	// ── Step 5: header / footer detection ────────────────────────────────────
	headers, footers := p.hfDetector.Detect(pages)
	log.Printf("[pdfpipeline] step=hf_detect headers=%d footers=%d", len(headers), len(footers))

	// ── Step 6: apply header/footer filter ───────────────────────────────────
	for i, page := range pages {
		filtered := FilterHeadersFooters(page.CleanText, headers, footers)
		if filtered != page.CleanText {
			pages[i].CleanText = filtered
			pages[i].Blocks = segmentBlocks(filtered)
		}
	}

	// ── Step 7: per-page quality filter ──────────────────────────────────────
	// Pages with fewer than minUsefulPageChars clean characters are likely
	// design/cover pages and would pollute the LLM context. OCR-sourced pages
	// are always kept because sparse text on a real scanned page is still
	// meaningful (e.g. a signature page).
	const minUsefulPageChars = 30
	var filteredPages []PageResult
	var skippedPageCount int
	for _, page := range pages {
		charCount := len([]rune(strings.TrimSpace(page.CleanText)))
		if charCount >= minUsefulPageChars || page.SourceType == PageSourceScannedImage {
			filteredPages = append(filteredPages, page)
		} else {
			skippedPageCount++
			log.Printf("[pdfpipeline] page=%d skipped (only %d chars after cleaning, source=%s)",
				page.PageNumber, charCount, page.SourceType)
		}
	}
	if skippedPageCount > 0 && len(filteredPages) > 0 {
		pages = filteredPages
		warnings = append(warnings, fmt.Sprintf("%d low-content page(s) skipped (design/cover pages)", skippedPageCount))
	}

	// ── Step 8: emit aggregated cleaning warnings ─────────────────────────────
	if len(pagesWithControlChars) > 0 {
		warnings = append(warnings, fmt.Sprintf("control characters removed from %d page(s)", len(pagesWithControlChars)))
	}
	if len(pagesWithWatermarks) > 0 {
		warnings = append(warnings, fmt.Sprintf("watermark text removed from %d page(s)", len(pagesWithWatermarks)))
	}
	if len(pagesWithNoise) > 0 {
		warnings = append(warnings, fmt.Sprintf("design extraction noise removed from %d page(s)", len(pagesWithNoise)))
	}

	// ── Step 9: aggregate final text ─────────────────────────────────────────
	aggregated := aggregatePages(pages)
	stats.TotalChars = len([]rune(aggregated))
	stats.ProcessingTimeMs = time.Since(start).Milliseconds()

	log.Printf("[pdfpipeline] done doc_id=%s pages=%d total_chars=%d ocr_used=%t duration_ms=%d",
		docID, len(pages), stats.TotalChars, ocrUsed, stats.ProcessingTimeMs)

	return &DocumentExtractionResult{
		DocumentID:     docID,
		FileName:       filepath.Base(filePath),
		Pages:          pages,
		AggregatedText: aggregated,
		Metadata: ExtractionMetadata{
			OCRUsed:          ocrUsed,
			HeaderCandidates: headers,
			FooterCandidates: footers,
			Warnings:         dedupeStrings(warnings),
			Stats:            stats,
		},
		ExtractedAt: time.Now().UTC(),
	}, nil
}

// ── helpers ──────────────────────────────────────────────────────────────────


func dedupeStrings(ss []string) []string {
	seen := make(map[string]bool, len(ss))
	out := make([]string, 0, len(ss))
	for _, s := range ss {
		s = strings.TrimSpace(s)
		if s != "" && !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}
