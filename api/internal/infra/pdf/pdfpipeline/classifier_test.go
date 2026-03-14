package pdfpipeline

import (
	"strings"
	"testing"
)

func TestClassify_NativeText(t *testing.T) {
	c := NewPageClassifier()

	// A page with plenty of printable text should be classified as native_text.
	text := strings.Repeat("contrato de prestação de serviços ", 5) // >60 printable chars
	got := c.Classify(text)
	if got != PageSourceNativeText {
		t.Errorf("expected %s, got %s", PageSourceNativeText, got)
	}
}

func TestClassify_ScannedImage(t *testing.T) {
	c := NewPageClassifier()

	// Empty page — no text layer at all.
	got := c.Classify("")
	if got != PageSourceScannedImage {
		t.Errorf("expected %s, got %s", PageSourceScannedImage, got)
	}

	got = c.Classify("   \n\t  ")
	if got != PageSourceScannedImage {
		t.Errorf("whitespace-only: expected %s, got %s", PageSourceScannedImage, got)
	}
}

func TestClassify_Unknown(t *testing.T) {
	c := NewPageClassifier()

	// A page with a small number of printable chars — unclear origin.
	got := c.Classify("Contrato") // 8 printable chars (below native threshold, above 0)
	if got != PageSourceUnknown {
		t.Errorf("expected %s, got %s", PageSourceUnknown, got)
	}
}

func TestCountPrintableRunes(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"abc", 3},
		{"a b c", 3},
		{"", 0},
		{"  \n\t  ", 0},
		{"abc\ndef", 6},
	}
	for _, tc := range tests {
		got := countPrintableRunes(tc.input)
		if got != tc.want {
			t.Errorf("countPrintableRunes(%q) = %d, want %d", tc.input, got, tc.want)
		}
	}
}
