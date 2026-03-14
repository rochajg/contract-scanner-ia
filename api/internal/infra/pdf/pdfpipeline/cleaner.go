package pdfpipeline

import (
	"strings"
	"unicode"
)

// CleaningStats records what the cleaner removed or changed.
type CleaningStats struct {
	InputLines       int
	OutputLines      int
	RemovedWatermark int
	RemovedDuplicate int
	HadControlChars  bool
	RemovedNoise     int // lines removed as design/extraction noise (no spaces, >50 chars)
}

// DocumentCleaner cleans and normalises raw extracted text.
type DocumentCleaner interface {
	Clean(raw string) (clean string, stats CleaningStats)
}

type defaultCleaner struct {
	watermarkPatterns []string
}

// NewDocumentCleaner returns a DocumentCleaner with built-in watermark patterns.
func NewDocumentCleaner() DocumentCleaner {
	return &defaultCleaner{
		watermarkPatterns: []string{
			"d4sign",
			"documento assinado eletronicamente",
			"secure.d4sign.com.br",
			"para confirmar as assinaturas acesse",
			"certificado de assinaturas gerado",
			"sincronizado com o ntp.br",
		},
	}
}

// Clean sanitises raw text by:
//  1. Removing invalid control characters (preserving \n, \r, \t).
//  2. Normalising each line (collapse internal whitespace).
//  3. Removing known watermark lines.
//  4. Collapsing overly-repeated lines.
func (c *defaultCleaner) Clean(raw string) (string, CleaningStats) {
	stats := CleaningStats{}
	if strings.TrimSpace(raw) == "" {
		return "", stats
	}

	sanitized, ctrlCount := removeControlChars(raw)
	stats.HadControlChars = ctrlCount > 0

	lines := strings.Split(sanitized, "\n")
	stats.InputLines = len(lines)

	repeats := make(map[string]int, len(lines))
	out := make([]string, 0, len(lines))

	for _, line := range lines {
		line = normalizeLine(line)
		if line == "" {
			continue
		}
		if c.isWatermark(line) {
			stats.RemovedWatermark++
			continue
		}
		// Remove lines that are extraction noise from design-heavy pages:
		// character-by-character positioned text in PDFs produces very long
		// lines with no spaces. Short lines without spaces ("CONTRATO", "R$",
		// "2.1") are valid legal content and must be kept.
		if isDesignNoiseLine(line) {
			stats.RemovedNoise++
			continue
		}
		if repeats[line] >= maxLineRepeats(line) {
			stats.RemovedDuplicate++
			continue
		}
		repeats[line]++
		out = append(out, line)
	}

	stats.OutputLines = len(out)
	return strings.TrimSpace(strings.Join(out, "\n")), stats
}

func (c *defaultCleaner) isWatermark(line string) bool {
	lower := strings.ToLower(line)
	for _, pat := range c.watermarkPatterns {
		if strings.Contains(lower, pat) {
			return true
		}
	}
	return false
}

// segmentBlocks splits clean text into logical TextBlocks separated by blank lines.
// Each block is classified with a simple heuristic (title vs paragraph).
func segmentBlocks(cleanText string) []TextBlock {
	if strings.TrimSpace(cleanText) == "" {
		return nil
	}

	var blocks []TextBlock
	var buf strings.Builder

	flush := func() {
		text := strings.TrimSpace(buf.String())
		if text != "" {
			blocks = append(blocks, TextBlock{
				Type: classifyBlock(text),
				Text: text,
			})
		}
		buf.Reset()
	}

	for _, line := range strings.Split(cleanText, "\n") {
		if strings.TrimSpace(line) == "" {
			flush()
			continue
		}
		buf.WriteString(line)
		buf.WriteString("\n")
	}
	flush()

	return blocks
}

// classifyBlock applies simple heuristics to distinguish titles from paragraphs.
func classifyBlock(text string) BlockType {
	lines := strings.Split(strings.TrimSpace(text), "\n")
	if len(lines) != 1 {
		return BlockTypeParagraph
	}
	line := strings.TrimSpace(lines[0])
	runes := []rune(line)

	if len(runes) > 80 {
		return BlockTypeParagraph
	}
	// ALL-CAPS single line (at least 2 runes) is treated as a title.
	if len(runes) >= 2 && strings.ToUpper(line) == line && strings.ContainsAny(line, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") {
		return BlockTypeTitle
	}
	// Lines ending with ":" are section headers.
	if strings.HasSuffix(line, ":") {
		return BlockTypeTitle
	}
	return BlockTypeParagraph
}

// removeControlChars strips Unicode control characters except \n, \r, \t.
func removeControlChars(input string) (string, int) {
	removed := 0
	out := strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r == '\t' {
			return r
		}
		if unicode.IsControl(r) {
			removed++
			return -1
		}
		return r
	}, input)
	return out, removed
}

// normalizeLine collapses internal whitespace in a single line.
func normalizeLine(line string) string {
	parts := strings.Fields(strings.TrimSpace(line))
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, " ")
}

// isDesignNoiseLine returns true for lines that are almost certainly extraction
// artifacts from design-heavy PDF pages (e.g. cover pages where text is
// positioned character-by-character without space markers). The rule is:
//   - the line has no spaces at all, AND
//   - it is longer than 50 runes.
//
// Short spaceless tokens like "CONTRATO", "R$", or "2.1" are valid legal text
// and are explicitly allowed through by the length guard.
func isDesignNoiseLine(line string) bool {
	if strings.Contains(line, " ") {
		return false
	}
	return len([]rune(line)) > 50
}

// maxLineRepeats returns how many times the same line may appear before being collapsed.
// Short lines are collapsed more aggressively (page numbers, bullet chars, etc.).
func maxLineRepeats(line string) int {
	r := []rune(strings.TrimSpace(line))
	switch {
	case len(r) <= 40:
		return 1
	case len(r) <= 120:
		return 2
	default:
		return 3
	}
}
