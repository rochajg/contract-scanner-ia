package providers

import "context"

// PageInfo summarises the extraction result for a single PDF page.
type PageInfo struct {
	PageNumber int    `json:"page_number"`
	SourceType string `json:"source_type"` // "native_text" | "scanned_image" | "hybrid" | "unknown"
	CharCount  int    `json:"char_count"`
}

// ExtractResult is the output of a PDF extraction.
// The first four fields are always populated.
// Pages, HeaderCandidates, FooterCandidates and TotalPages are optional enrichments
// produced by the richer PipelineExtractor; they are zero-valued when the legacy
// HybridExtractor is used.
type ExtractResult struct {
	Text             string
	Source           string  // document-level: "native" | "ocr" | "hybrid"
	QualityScore     float64 // 0–1
	Warnings         []string
	Pages            []PageInfo // per-page summary (optional)
	HeaderCandidates []string   // detected header lines (optional)
	FooterCandidates []string   // detected footer lines (optional)
	TotalPages       int        // total page count (optional)
}

type PDFExtractor interface {
	Extract(ctx context.Context, filePath string) (*ExtractResult, error)
}
