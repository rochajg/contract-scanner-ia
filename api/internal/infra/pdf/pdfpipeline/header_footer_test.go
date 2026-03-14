package pdfpipeline

import (
	"slices"
	"testing"
)

// buildPages is a test helper that creates PageResult slices from clean text strings.
func buildPages(texts ...string) []PageResult {
	pages := make([]PageResult, len(texts))
	for i, t := range texts {
		pages[i] = PageResult{PageNumber: i + 1, CleanText: t}
	}
	return pages
}

func TestDetect_ReturnsNothingForFewPages(t *testing.T) {
	d := NewHeaderFooterDetector()
	pages := buildPages("Contrato entre as partes", "Cláusula segunda")
	// minPages=2 but minFreqRatio=0.4 → need 2*0.4=0.8 → ceil to 2 appearances
	// The line "Contrato" appears only on page 1 — should not be detected.
	h, f := d.Detect(pages)
	if len(h) != 0 || len(f) != 0 {
		t.Errorf("expected no candidates for pages with no repetition; h=%v f=%v", h, f)
	}
}

func TestDetect_RepeatedLineIsDetected(t *testing.T) {
	d := NewHeaderFooterDetector()

	const hdr = "Contrato de Prestação de Serviços"
	// Repeat the same header on 5 pages; add unique content per page.
	pages := buildPages(
		hdr+"\nTexto contratual da cláusula primeira.",
		hdr+"\nTexto contratual da cláusula segunda.",
		hdr+"\nTexto contratual da cláusula terceira.",
		hdr+"\nTexto contratual da cláusula quarta.",
		hdr+"\nTexto contratual da cláusula quinta.",
	)

	h, _ := d.Detect(pages)
	if !slices.Contains(h, hdr) {
		t.Errorf("expected %q in header candidates, got %v", hdr, h)
	}
}

func TestDetect_FooterByPosition(t *testing.T) {
	d := NewHeaderFooterDetector()

	const ftr = "Página 1 de 5"
	// Put the footer at the END of each page.
	pages := buildPages(
		"Linha um\nLinha dois\nLinha tres\nLinha quatro\n"+ftr,
		"Linha A\nLinha B\nLinha C\nLinha D\n"+ftr,
		"Linha I\nLinha II\nLinha III\nLinha IV\n"+ftr,
		"Linha X\nLinha XI\nLinha XII\nLinha XIII\n"+ftr,
		"Linha AA\nLinha BB\nLinha CC\nLinha DD\n"+ftr,
	)

	_, f := d.Detect(pages)
	if !slices.Contains(f, ftr) {
		t.Errorf("expected %q in footer candidates, got %v", ftr, f)
	}
}

func TestFilterHeadersFooters_RemovesLines(t *testing.T) {
	headers := []string{"HEADER LINE"}
	footers := []string{"footer line"}

	text := "HEADER LINE\nSome contract text\nfooter line"
	got := FilterHeadersFooters(text, headers, footers)

	if got != "Some contract text" {
		t.Errorf("unexpected result: %q", got)
	}
}

func TestFilterHeadersFooters_NoOp(t *testing.T) {
	text := "line one\nline two"
	got := FilterHeadersFooters(text, nil, nil)
	if got != text {
		t.Errorf("expected unchanged text, got %q", got)
	}
}

func TestNonEmptyLines(t *testing.T) {
	lines := nonEmptyLines("a\n\nb\n\n\nc")
	if len(lines) != 3 {
		t.Errorf("expected 3 non-empty lines, got %d: %v", len(lines), lines)
	}
}
