package pdfpipeline

import (
	"sort"
	"strings"
)

// HeaderFooterDetector identifies repeated lines that are likely headers or footers.
type HeaderFooterDetector interface {
	// Detect returns header and footer candidate lines found across pages.
	Detect(pages []PageResult) (headers []string, footers []string)
}

type repetitionDetector struct {
	minPages     int     // minimum number of pages a line must appear on
	minFreqRatio float64 // minimum ratio of page appearances / total pages
}

// NewHeaderFooterDetector returns a detector based on cross-page line repetition.
// A line is a header/footer candidate when it appears on at least 40% of pages
// (and on at least 2 pages). Position within the page determines header vs footer.
func NewHeaderFooterDetector() HeaderFooterDetector {
	return &repetitionDetector{
		minPages:     2,
		minFreqRatio: 0.4,
	}
}

// Detect scans all pages for lines that repeat frequently and classifies them
// as headers (appear near the top of the page) or footers (near the bottom).
//
// The function intentionally avoids deleting text just because it repeats —
// candidates are returned separately so callers can apply them selectively.
func (d *repetitionDetector) Detect(pages []PageResult) ([]string, []string) {
	if len(pages) < d.minPages {
		return nil, nil
	}

	type lineInfo struct {
		count    int // total pages the line appears on
		topCount int // pages where the line is in the top 20% of lines
		botCount int // pages where the line is in the bottom 20% of lines
	}

	freq := make(map[string]*lineInfo)

	for _, page := range pages {
		lines := nonEmptyLines(page.CleanText)
		n := len(lines)
		if n == 0 {
			continue
		}
		topN := max(1, n/5)
		botN := max(1, n/5)

		seenOnPage := make(map[string]bool)
		for i, line := range lines {
			line = strings.TrimSpace(line)
			if len([]rune(line)) < 5 || seenOnPage[line] {
				continue
			}
			seenOnPage[line] = true

			if freq[line] == nil {
				freq[line] = &lineInfo{}
			}
			info := freq[line]
			info.count++
			if i < topN {
				info.topCount++
			}
			if i >= n-botN {
				info.botCount++
			}
		}
	}

	totalPages := len(pages)
	minCount := max(d.minPages, int(float64(totalPages)*d.minFreqRatio))

	// Sort for deterministic output.
	keys := make([]string, 0, len(freq))
	for k := range freq {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var headers, footers []string
	for _, line := range keys {
		info := freq[line]
		if info.count < minCount {
			continue
		}
		if info.topCount >= info.botCount {
			headers = append(headers, line)
		} else {
			footers = append(footers, line)
		}
	}
	return headers, footers
}

// FilterHeadersFooters removes known header/footer lines from a text string.
// Lines are matched exactly (after trimming whitespace).
func FilterHeadersFooters(text string, headers, footers []string) string {
	if len(headers) == 0 && len(footers) == 0 {
		return text
	}
	blacklist := make(map[string]bool, len(headers)+len(footers))
	for _, h := range headers {
		blacklist[strings.TrimSpace(h)] = true
	}
	for _, f := range footers {
		blacklist[strings.TrimSpace(f)] = true
	}

	lines := strings.Split(text, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if !blacklist[strings.TrimSpace(line)] {
			out = append(out, line)
		}
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

// nonEmptyLines returns only the non-blank lines from text.
func nonEmptyLines(text string) []string {
	raw := strings.Split(text, "\n")
	result := make([]string, 0, len(raw))
	for _, line := range raw {
		if strings.TrimSpace(line) != "" {
			result = append(result, line)
		}
	}
	return result
}
