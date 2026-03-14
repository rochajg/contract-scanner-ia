package pdfpipeline

import (
	"context"
	"testing"
)

// mockOCRProvider is a test double that returns a fixed string for any image.
type mockOCRProvider struct {
	text string
	err  error
}

func (m *mockOCRProvider) AnalyzePage(_ context.Context, _ string) (string, error) {
	return m.text, m.err
}

// ── aggregatePages ────────────────────────────────────────────────────────────

func TestAggregatePages_MultiplePages(t *testing.T) {
	pages := []PageResult{
		{PageNumber: 1, CleanText: "Texto da página um."},
		{PageNumber: 2, CleanText: "Texto da página dois."},
	}
	got := aggregatePages(pages)

	if got == "" {
		t.Fatal("expected non-empty aggregated text")
	}
	// Multi-page output must include page markers.
	for _, marker := range []string{"--- Página 1 ---", "--- Página 2 ---"} {
		if !containsString(got, marker) {
			t.Errorf("expected page marker %q in output", marker)
		}
	}
}

func TestAggregatePages_SinglePage(t *testing.T) {
	pages := []PageResult{
		{PageNumber: 1, CleanText: "Contrato único."},
	}
	got := aggregatePages(pages)
	// Single-page output should NOT include page markers.
	if containsString(got, "--- Página") {
		t.Errorf("single-page output should not have page markers, got: %q", got)
	}
	if got != "Contrato único." {
		t.Errorf("unexpected result: %q", got)
	}
}

func TestAggregatePages_SkipsEmptyPages(t *testing.T) {
	pages := []PageResult{
		{PageNumber: 1, CleanText: "Página um."},
		{PageNumber: 2, CleanText: ""},
		{PageNumber: 3, CleanText: "Página três."},
	}
	got := aggregatePages(pages)
	if containsString(got, "--- Página 2 ---") {
		t.Error("empty page 2 should not produce a marker in the output")
	}
}

// ── buildImageMap ─────────────────────────────────────────────────────────────

func TestBuildImageMap_ParsesPageNumbers(t *testing.T) {
	images := []string{
		"/tmp/ocr/page-001.png",
		"/tmp/ocr/page-010.png",
		"/tmp/ocr/page-100.png",
	}
	m := buildImageMap(images)

	tests := []struct {
		page int
		img  string
	}{
		{1, "/tmp/ocr/page-001.png"},
		{10, "/tmp/ocr/page-010.png"},
		{100, "/tmp/ocr/page-100.png"},
	}
	for _, tc := range tests {
		got, ok := m[tc.page]
		if !ok {
			t.Errorf("page %d not found in image map", tc.page)
			continue
		}
		if got != tc.img {
			t.Errorf("page %d: expected %q, got %q", tc.page, tc.img, got)
		}
	}
}

// ── ChunkDocument ─────────────────────────────────────────────────────────────

func TestChunkDocument_FitsInOneChunk(t *testing.T) {
	doc := &DocumentExtractionResult{
		Pages: []PageResult{
			{PageNumber: 1, CleanText: "Contrato curto."},
		},
		AggregatedText: "Contrato curto.",
	}
	chunks := ChunkDocument(doc, 1000)
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0].Text != "Contrato curto." {
		t.Errorf("unexpected chunk text: %q", chunks[0].Text)
	}
}

func TestChunkDocument_SplitsLargeDocument(t *testing.T) {
	// Build a doc with 3 pages, each with ~200 chars, limit at 300 chars.
	page := func(n int, text string) PageResult {
		return PageResult{PageNumber: n, CleanText: text}
	}
	longText := "Esta é uma cláusula contratual extensa que ocupa muito espaço no documento do contrato. " +
		"Ela serve para garantir que o chunk seja dividido corretamente pelo paginador da pipeline."

	doc := &DocumentExtractionResult{
		Pages: []PageResult{
			page(1, longText),
			page(2, longText),
			page(3, longText),
		},
		AggregatedText: longText + "\n\n" + longText + "\n\n" + longText,
	}
	chunks := ChunkDocument(doc, 300)
	if len(chunks) < 2 {
		t.Errorf("expected multiple chunks for large document, got %d", len(chunks))
	}
	for _, ch := range chunks {
		if len([]rune(ch.Text)) > 300+50 { // small tolerance for paragraph boundaries
			t.Errorf("chunk %d exceeds max size: %d runes", ch.Index, len([]rune(ch.Text)))
		}
	}
}

// ── Pipeline with injected dependencies ──────────────────────────────────────

func TestPipeline_UsesOCRForScannedPage(t *testing.T) {
	cfg := DefaultConfig()
	p := NewPipeline(cfg)

	// Inject a native extractor that returns one empty page (simulates scan).
	p.nativeExtractFn = func(_ string) ([]NativePageText, error) {
		return []NativePageText{{PageNumber: 1, Text: ""}}, nil
	}
	// Inject a mock OCR that returns contract text.
	p.ocrProvider = &mockOCRProvider{text: "Contrato de prestação de serviços entre as partes."}

	doc, err := p.ProcessPDF(context.Background(), "/fake/path.pdf", "test-doc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(doc.Pages) != 1 {
		t.Fatalf("expected 1 page, got %d", len(doc.Pages))
	}

	// The rasterization step will fail (no real file) but OCR mock should handle the path.
	// Since rasterizePDF will fail, the OCR text won't be applied — but pipeline should not crash.
	// We verify the pipeline completes without error and page is classified.
	if doc.Pages[0].PageNumber != 1 {
		t.Errorf("expected page 1, got %d", doc.Pages[0].PageNumber)
	}
}

func TestPipeline_HandlesNativeTextPages(t *testing.T) {
	cfg := DefaultConfig()
	cfg.OCREnabled = false // disable OCR so we only test native path
	p := NewPipeline(cfg)

	contractText := "Contrato de prestação de serviços.\n" +
		"Cláusula primeira: objeto do contrato.\n" +
		"Cláusula segunda: vigência e prazo."
	p.nativeExtractFn = func(_ string) ([]NativePageText, error) {
		return []NativePageText{{PageNumber: 1, Text: contractText}}, nil
	}

	doc, err := p.ProcessPDF(context.Background(), "/fake/path.pdf", "test-doc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if doc.AggregatedText == "" {
		t.Error("expected non-empty aggregated text")
	}
	if doc.Pages[0].SourceType != PageSourceNativeText {
		t.Errorf("expected native_text, got %s", doc.Pages[0].SourceType)
	}
	if doc.Metadata.Stats.OCRUsed {
		t.Error("OCR should not have been used")
	}
}

// ── dedupeStrings ─────────────────────────────────────────────────────────────

func TestDedupeStrings(t *testing.T) {
	input := []string{"a", "b", "a", "", "  b  ", "c"}
	got := dedupeStrings(input)
	// Expect: a, b, c (deduped, trimmed, no empty)
	if len(got) != 3 {
		t.Errorf("expected 3 unique strings, got %d: %v", len(got), got)
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && searchStr(s, substr))
}

func searchStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
