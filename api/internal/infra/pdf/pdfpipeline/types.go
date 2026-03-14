package pdfpipeline

import "time"

// PageSourceType classifies how the text for a given page was obtained.
type PageSourceType string

const (
	PageSourceNativeText   PageSourceType = "native_text"
	PageSourceScannedImage PageSourceType = "scanned_image"
	PageSourceHybrid       PageSourceType = "hybrid"
	PageSourceUnknown      PageSourceType = "unknown"
)

// BlockType classifies a text block within a page.
type BlockType string

const (
	BlockTypeParagraph BlockType = "paragraph"
	BlockTypeTitle     BlockType = "title"
	BlockTypeHeader    BlockType = "header"
	BlockTypeFooter    BlockType = "footer"
	BlockTypeUnknown   BlockType = "unknown"
)

// TextBlock is a logical unit of text within a page.
type TextBlock struct {
	Type       BlockType `json:"type"`
	Text       string    `json:"text"`
	Confidence float64   `json:"confidence,omitempty"`
}

// PageResult holds extraction results for a single PDF page.
type PageResult struct {
	PageNumber int            `json:"page_number"`
	SourceType PageSourceType `json:"source_type"`
	RawText    string         `json:"raw_text,omitempty"`
	CleanText  string         `json:"clean_text"`
	Blocks     []TextBlock    `json:"blocks,omitempty"`
}

// ProcessingStats tracks what happened during the pipeline run.
type ProcessingStats struct {
	TotalPages       int   `json:"total_pages"`
	NativePages      int   `json:"native_pages"`
	OCRPages         int   `json:"ocr_pages"`
	HybridPages      int   `json:"hybrid_pages"`
	UnknownPages     int   `json:"unknown_pages"`
	TotalChars       int   `json:"total_chars"`
	ProcessingTimeMs int64 `json:"processing_time_ms"`
	OCRUsed          bool  `json:"ocr_used"`
}

// ExtractionMetadata holds audit information about the extraction process.
type ExtractionMetadata struct {
	OCRUsed          bool            `json:"ocr_used"`
	HeaderCandidates []string        `json:"header_candidates,omitempty"`
	FooterCandidates []string        `json:"footer_candidates,omitempty"`
	Warnings         []string        `json:"warnings,omitempty"`
	Stats            ProcessingStats `json:"stats"`
}

// DocumentExtractionResult is the rich, page-level output of the PDF pipeline.
// It is serialisable to JSON for debugging and audit purposes.
type DocumentExtractionResult struct {
	DocumentID     string             `json:"document_id"`
	FileName       string             `json:"file_name"`
	Pages          []PageResult       `json:"pages"`
	AggregatedText string             `json:"aggregated_text"`
	Metadata       ExtractionMetadata `json:"metadata"`
	ExtractedAt    time.Time          `json:"extracted_at"`
}
