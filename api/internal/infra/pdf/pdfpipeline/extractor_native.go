package pdfpipeline

import (
	"fmt"

	gopdf "github.com/ledongthuc/pdf"
)

// NativePageText holds the raw text extracted from one page via the PDF text layer.
type NativePageText struct {
	PageNumber int
	Text       string
	Err        error // non-nil when extraction failed for this specific page
}

// extractAllPagesNative opens a PDF and extracts raw text from every page using
// the embedded text layer. It never fails mid-stream: per-page errors are captured
// in NativePageText.Err so callers can decide how to handle them.
func extractAllPagesNative(filePath string) ([]NativePageText, error) {
	f, reader, err := gopdf.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("opening pdf: %w", err)
	}
	defer f.Close()

	numPages := reader.NumPage()
	results := make([]NativePageText, numPages)

	for i := 1; i <= numPages; i++ {
		page := reader.Page(i)
		if page.V.IsNull() {
			results[i-1] = NativePageText{PageNumber: i}
			continue
		}
		text, pageErr := page.GetPlainText(nil)
		results[i-1] = NativePageText{
			PageNumber: i,
			Text:       text,
			Err:        pageErr,
		}
	}

	return results, nil
}
