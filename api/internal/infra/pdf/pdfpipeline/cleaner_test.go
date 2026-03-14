package pdfpipeline

import (
	"strings"
	"testing"
)

func TestClean_RemovesWatermarks(t *testing.T) {
	c := NewDocumentCleaner()

	raw := "Contrato de serviços\nd4sign\nCláusula primeira"
	got, stats := c.Clean(raw)

	if strings.Contains(got, "d4sign") {
		t.Errorf("watermark line should have been removed, got: %q", got)
	}
	if stats.RemovedWatermark != 1 {
		t.Errorf("expected RemovedWatermark=1, got %d", stats.RemovedWatermark)
	}
}

func TestClean_RemovesControlChars(t *testing.T) {
	c := NewDocumentCleaner()

	// \x01 and \x02 are control chars; \n should be preserved.
	raw := "linha um\x01\nliga dois\x02"
	got, stats := c.Clean(raw)

	if strings.ContainsAny(got, "\x01\x02") {
		t.Errorf("control characters should have been removed, got: %q", got)
	}
	if !stats.HadControlChars {
		t.Error("expected HadControlChars=true")
	}
}

func TestClean_CollapsesDuplicateShortLines(t *testing.T) {
	c := NewDocumentCleaner()

	// A short line (≤40 chars) repeated twice should be collapsed to one occurrence.
	raw := "Página 1\nConteúdo do contrato\nPágina 1"
	got, stats := c.Clean(raw)

	occurrences := strings.Count(got, "Página 1")
	if occurrences > 1 {
		t.Errorf("short duplicate line should be collapsed to 1, got %d occurrences", occurrences)
	}
	if stats.RemovedDuplicate < 1 {
		t.Errorf("expected at least 1 removed duplicate, got %d", stats.RemovedDuplicate)
	}
}

func TestClean_PreservesLongDuplicates(t *testing.T) {
	c := NewDocumentCleaner()

	// A line longer than 120 chars may appear up to 3 times.
	longLine := strings.Repeat("Esta é uma cláusula contratual bastante longa que descreve condições ", 3)
	raw := longLine + "\n" + longLine
	got, _ := c.Clean(raw)

	if strings.TrimSpace(got) == "" {
		t.Error("long lines should not all be removed")
	}
}

func TestClean_EmptyInput(t *testing.T) {
	c := NewDocumentCleaner()

	got, stats := c.Clean("")
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
	if stats.InputLines != 0 {
		t.Errorf("expected InputLines=0, got %d", stats.InputLines)
	}
}

func TestSegmentBlocks_BasicParagraphs(t *testing.T) {
	text := "Primeira linha\nSegunda linha\n\nTerceira linha\nQuarta linha"
	blocks := segmentBlocks(text)

	if len(blocks) != 2 {
		t.Errorf("expected 2 blocks, got %d", len(blocks))
	}
	for _, b := range blocks {
		if b.Type != BlockTypeParagraph && b.Type != BlockTypeTitle {
			t.Errorf("unexpected block type: %s", b.Type)
		}
	}
}

func TestSegmentBlocks_TitleDetection(t *testing.T) {
	// ALL-CAPS single line should be classified as title.
	blocks := segmentBlocks("CLÁUSULA PRIMEIRA\n\nO contrato tem vigência de 12 meses.")
	if len(blocks) < 1 {
		t.Fatal("expected at least 1 block")
	}
	if blocks[0].Type != BlockTypeTitle {
		t.Errorf("ALL-CAPS line should be BlockTypeTitle, got %s", blocks[0].Type)
	}
}

func TestSegmentBlocks_EmptyInput(t *testing.T) {
	blocks := segmentBlocks("")
	if blocks != nil {
		t.Errorf("expected nil for empty input, got %v", blocks)
	}
}

func TestMaxLineRepeats(t *testing.T) {
	tests := []struct {
		line string
		want int
	}{
		{strings.Repeat("x", 10), 1},  // short ≤40 → 1
		{strings.Repeat("x", 80), 2},  // medium ≤120 → 2
		{strings.Repeat("x", 150), 3}, // long >120 → 3
	}
	for _, tc := range tests {
		got := maxLineRepeats(tc.line)
		if got != tc.want {
			t.Errorf("maxLineRepeats(len=%d) = %d, want %d", len(tc.line), got, tc.want)
		}
	}
}
