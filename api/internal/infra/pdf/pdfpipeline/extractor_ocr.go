package pdfpipeline

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// OCRProvider defines the contract for OCR processing of a pre-rendered page image.
// Implementations are swappable: use TesseractProvider locally or a cloud OCR in production.
type OCRProvider interface {
	// AnalyzePage runs OCR on the image at imagePath and returns the extracted text.
	AnalyzePage(ctx context.Context, imagePath string) (string, error)
}

// TesseractProvider implements OCRProvider using the system `tesseract` binary.
type TesseractProvider struct {
	lang string
}

// NewTesseractProvider returns an OCRProvider backed by tesseract.
// lang follows tesseract language codes, e.g. "por+eng".
func NewTesseractProvider(lang string) OCRProvider {
	if lang == "" {
		lang = "por+eng"
	}
	return &TesseractProvider{lang: lang}
}

// AnalyzePage runs tesseract on the given image file and returns the raw OCR text.
func (t *TesseractProvider) AnalyzePage(ctx context.Context, imagePath string) (string, error) {
	out, err := exec.CommandContext(
		ctx, "tesseract", imagePath, "stdout", "-l", t.lang, "--psm", "6",
	).CombinedOutput()
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return "", fmt.Errorf("tesseract timed out on %s", filepath.Base(imagePath))
		}
		return "", fmt.Errorf("tesseract failed on %s: %w (%s)",
			filepath.Base(imagePath), err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

// rasterizePDF converts PDF pages to PNG images using pdftoppm.
// When maxPages <= 0 all pages are converted; otherwise pages 1..maxPages are converted.
// Returns image paths sorted by page order.
func rasterizePDF(ctx context.Context, filePath, targetDir string, dpi, maxPages int) ([]string, error) {
	prefix := filepath.Join(targetDir, "page")
	args := []string{"-r", strconv.Itoa(dpi), "-png"}
	if maxPages > 0 {
		args = append(args, "-f", "1", "-l", strconv.Itoa(maxPages))
	}
	args = append(args, filePath, prefix)

	log.Printf("[pdfpipeline] pdftoppm args=%v", args)
	if out, err := exec.CommandContext(ctx, "pdftoppm", args...).CombinedOutput(); err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, fmt.Errorf("pdftoppm timed out")
		}
		return nil, fmt.Errorf("pdftoppm: %w (%s)", err, strings.TrimSpace(string(out)))
	}

	images, err := filepath.Glob(filepath.Join(targetDir, "page-*.png"))
	if err != nil {
		return nil, fmt.Errorf("listing rasterized images: %w", err)
	}
	if len(images) == 0 {
		return nil, fmt.Errorf("pdftoppm produced no images")
	}
	sort.Strings(images)
	return images, nil
}

// buildImageMap maps PDF page number → image file path from a sorted slice of
// pdftoppm output files named "page-001.png", "page-002.png", etc.
func buildImageMap(images []string) map[int]string {
	m := make(map[int]string, len(images))
	for _, img := range images {
		base := filepath.Base(img)
		name := strings.TrimPrefix(base, "page-")
		name = strings.TrimSuffix(name, ".png")
		pageNum, err := strconv.Atoi(name)
		if err == nil && pageNum > 0 {
			m[pageNum] = img
		}
	}
	return m
}
