package pdfreader

import (
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
)

const (
	minLineOverlap    = 0.5
	maxWordGapRatio   = 1.5
	paragraphGapRatio = 1.4
	maxLetterGapEm    = 0.35
)

type textRun struct {
	text                     string
	left, top, right, bottom float64
	em                       float64
}

func (r textRun) height() float64 { return r.bottom - r.top }

func (r textRun) centerY() float64 { return (r.top + r.bottom) / 2 }

type lineBand struct {
	top, bottom float64
	runs        []textRun
}

func (b *lineBand) accepts(run textRun) bool {
	overlap := min(b.bottom, run.bottom) - max(b.top, run.top)
	shorter := min(b.bottom-b.top, run.height())
	if shorter <= 0 {
		return run.centerY() >= b.top && run.centerY() <= b.bottom
	}
	return overlap/shorter >= minLineOverlap
}

func (b *lineBand) add(run textRun) {
	b.top = min(b.top, run.top)
	b.bottom = max(b.bottom, run.bottom)
	b.runs = append(b.runs, run)
}

func wordsOf(glyphs []textRun) []textRun {
	words := make([]textRun, 0, len(glyphs)/4+1)
	var text strings.Builder
	var current textRun
	open := false
	flush := func() {
		if !open {
			return
		}
		current.text = text.String()
		words = append(words, current)
		text.Reset()
		open = false
	}

	for _, glyph := range glyphs {
		if strings.TrimSpace(glyph.text) == "" {
			flush()
			continue
		}
		if open && !current.continuesWith(glyph) {
			flush()
		}
		if open {
			current.left = min(current.left, glyph.left)
			current.top = min(current.top, glyph.top)
			current.right = max(current.right, glyph.right)
			current.bottom = max(current.bottom, glyph.bottom)
			current.em = max(current.em, glyph.em)
		} else {
			current = glyph
			open = true
		}
		text.WriteString(glyph.text)
	}
	flush()

	return words
}

func (r textRun) continuesWith(next textRun) bool {
	em := max(r.em, next.em, r.height(), next.height())
	if em <= 0 {
		return false
	}
	apartX := max(next.left-r.right, r.left-next.right, 0)
	apartY := max(next.top-r.bottom, r.top-next.bottom, 0)

	return apartX <= em*maxLetterGapEm && apartY <= em*maxLetterGapEm
}

func arrange(runs []textRun) (arranged []kernelports.PDFTextLine, readingOrder string) {
	if len(runs) == 0 {
		return nil, ""
	}

	sorted := slices.Clone(runs)
	slices.SortStableFunc(sorted, func(a, b textRun) int {
		switch {
		case a.centerY() < b.centerY():
			return -1
		case a.centerY() > b.centerY():
			return 1
		default:
			return 0
		}
	})

	bands := make([]*lineBand, 0, len(sorted))
	for _, run := range sorted {
		if n := len(bands); n > 0 && bands[n-1].accepts(run) {
			bands[n-1].add(run)
			continue
		}
		bands = append(bands, &lineBand{top: run.top, bottom: run.bottom, runs: []textRun{run}})
	}

	lines := make([]kernelports.PDFTextLine, 0, len(bands))
	var text strings.Builder
	for idx, band := range bands {
		segments := splitBand(band)
		lines = append(lines, segments...)
		if idx > 0 {
			text.WriteByte('\n')
			previous := bands[idx-1]
			if band.top-previous.bottom > (previous.bottom-previous.top)*paragraphGapRatio {
				text.WriteByte('\n')
			}
		}
		for i, segment := range segments {
			if i > 0 {
				text.WriteString(kernelports.PDFColumnGap)
			}
			text.WriteString(segment.Text)
		}
	}

	return lines, text.String()
}

func splitBand(band *lineBand) []kernelports.PDFTextLine {
	slices.SortStableFunc(band.runs, func(a, b textRun) int {
		switch {
		case a.left < b.left:
			return -1
		case a.left > b.left:
			return 1
		default:
			return 0
		}
	})

	maxGap := (band.bottom - band.top) * maxWordGapRatio

	var lines []kernelports.PDFTextLine
	var parts []string
	var current textRun
	flush := func() {
		if len(parts) == 0 {
			return
		}
		lines = append(lines, kernelports.PDFTextLine{
			Text:   strings.Join(parts, " "),
			Left:   current.left,
			Top:    current.top,
			Width:  current.right - current.left,
			Height: current.bottom - current.top,
		})
		parts = parts[:0]
	}

	for _, run := range band.runs {
		if len(parts) > 0 && run.left-current.right > maxGap {
			flush()
		}
		if len(parts) == 0 {
			current = run
		} else {
			current.left = min(current.left, run.left)
			current.top = min(current.top, run.top)
			current.right = max(current.right, run.right)
			current.bottom = max(current.bottom, run.bottom)
		}
		parts = append(parts, run.text)
	}
	flush()

	return lines
}
