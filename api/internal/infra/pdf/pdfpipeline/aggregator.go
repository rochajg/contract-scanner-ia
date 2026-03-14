package pdfpipeline

import (
	"fmt"
	"strings"
)

// aggregatePages combines the CleanText of all pages into a single document string.
// Pages with empty text are skipped. For multi-page documents a page marker is
// inserted so the AI can reason about page boundaries when needed.
func aggregatePages(pages []PageResult) string {
	var sb strings.Builder
	multiPage := countNonEmpty(pages) > 1

	for _, page := range pages {
		text := strings.TrimSpace(page.CleanText)
		if text == "" {
			continue
		}
		if sb.Len() > 0 {
			sb.WriteString("\n\n")
		}
		if multiPage {
			sb.WriteString(fmt.Sprintf("--- Página %d ---\n", page.PageNumber))
		}
		sb.WriteString(text)
	}
	return strings.TrimSpace(sb.String())
}

func countNonEmpty(pages []PageResult) int {
	n := 0
	for _, p := range pages {
		if strings.TrimSpace(p.CleanText) != "" {
			n++
		}
	}
	return n
}
