package pdfpipeline

import (
	"strings"
	"unicode"
)

const (
	minPrintableForNative = 60
	minPrintableForHybrid = 15
)

// PageClassifier determines the PageSourceType of a page from its native text.
// Classification happens before OCR; callers update SourceType after OCR if needed.
type PageClassifier interface {
	Classify(rawNativeText string) PageSourceType
}

type defaultPageClassifier struct{}

// NewPageClassifier returns a PageClassifier with default thresholds.
func NewPageClassifier() PageClassifier {
	return &defaultPageClassifier{}
}

// Classify returns the source type based on the amount of printable characters
// found in the natively-extracted page text.
//
//   - >= minPrintableForNative  → native_text  (text layer is present and useful)
//   - >= minPrintableForHybrid  → unknown      (some text but might need OCR too)
//   - 0                         → scanned_image (no text layer at all)
func (c *defaultPageClassifier) Classify(rawNativeText string) PageSourceType {
	printable := countPrintableRunes(strings.TrimSpace(rawNativeText))

	switch {
	case printable >= minPrintableForNative:
		return PageSourceNativeText
	case printable >= minPrintableForHybrid:
		return PageSourceUnknown
	case printable == 0:
		return PageSourceScannedImage
	default:
		return PageSourceUnknown
	}
}

// countPrintableRunes counts non-whitespace, non-control runes in s.
func countPrintableRunes(s string) int {
	n := 0
	for _, r := range s {
		if !unicode.IsControl(r) && !unicode.IsSpace(r) {
			n++
		}
	}
	return n
}
