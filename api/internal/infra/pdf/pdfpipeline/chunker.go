package pdfpipeline

import (
	"strings"
)

// Chunk is a portion of document text with page-level traceability.
type Chunk struct {
	Index    int    `json:"index"`
	Text     string `json:"text"`
	PageFrom int    `json:"page_from,omitempty"`
	PageTo   int    `json:"page_to,omitempty"`
	CharLen  int    `json:"char_len"`
}

// ChunkDocument splits a DocumentExtractionResult into chunks that fit within
// maxCharsPerChunk runes. Splitting prefers page boundaries; if a single page
// still exceeds the limit it is split at paragraph boundaries.
//
// When maxCharsPerChunk <= 0 a default of 12 000 is used.
// If the whole document fits in one chunk it is returned as-is.
func ChunkDocument(doc *DocumentExtractionResult, maxCharsPerChunk int) []Chunk {
	if maxCharsPerChunk <= 0 {
		maxCharsPerChunk = 12_000
	}

	aggregated := doc.AggregatedText
	if len([]rune(aggregated)) <= maxCharsPerChunk {
		return []Chunk{{
			Index:   0,
			Text:    aggregated,
			CharLen: len([]rune(aggregated)),
		}}
	}

	// Build page sections from the structured pages (preserving page numbers).
	sections := buildPageSections(doc.Pages)

	var chunks []Chunk
	var buf strings.Builder
	pageFrom, pageTo := firstPage(sections), firstPage(sections)

	flush := func() {
		text := strings.TrimSpace(buf.String())
		if text != "" {
			chunks = append(chunks, Chunk{
				Index:    len(chunks),
				Text:     text,
				PageFrom: pageFrom,
				PageTo:   pageTo,
				CharLen:  len([]rune(text)),
			})
		}
		buf.Reset()
	}

	for _, sec := range sections {
		secRunes := len([]rune(sec.text))

		if buf.Len() > 0 && len([]rune(buf.String()))+secRunes > maxCharsPerChunk {
			flush()
			pageFrom = sec.page
		}

		// If a single page is still larger than the limit, split it at paragraphs.
		if secRunes > maxCharsPerChunk {
			for _, part := range splitAtParagraphs(sec.text, maxCharsPerChunk) {
				if len([]rune(buf.String()))+len([]rune(part)) > maxCharsPerChunk && buf.Len() > 0 {
					flush()
					pageFrom = sec.page
				}
				buf.WriteString(part)
				buf.WriteString("\n\n")
				pageTo = sec.page
			}
		} else {
			buf.WriteString(sec.text)
			buf.WriteString("\n\n")
			pageTo = sec.page
		}
	}
	flush()

	return chunks
}

type pageSection struct {
	page int
	text string
}

func buildPageSections(pages []PageResult) []pageSection {
	sections := make([]pageSection, 0, len(pages))
	for _, p := range pages {
		if strings.TrimSpace(p.CleanText) != "" {
			sections = append(sections, pageSection{page: p.PageNumber, text: p.CleanText})
		}
	}
	return sections
}

func firstPage(sections []pageSection) int {
	if len(sections) == 0 {
		return 1
	}
	return sections[0].page
}

// splitAtParagraphs splits text at blank-line boundaries so that each part is
// smaller than maxRunes. Parts that are still too large are returned as-is.
func splitAtParagraphs(text string, maxRunes int) []string {
	paragraphs := strings.Split(text, "\n\n")
	var parts []string
	var buf strings.Builder

	for _, para := range paragraphs {
		paraRunes := len([]rune(para))
		if len([]rune(buf.String()))+paraRunes > maxRunes && buf.Len() > 0 {
			parts = append(parts, strings.TrimSpace(buf.String()))
			buf.Reset()
		}
		buf.WriteString(para)
		buf.WriteString("\n\n")
	}
	if buf.Len() > 0 {
		if part := strings.TrimSpace(buf.String()); part != "" {
			parts = append(parts, part)
		}
	}
	return parts
}
