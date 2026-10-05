package pdfreader

import (
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/core/ports/services"
)

const (
	minLineOverlap  = 0.5
	maxWordGapRatio = 1.5
)

type textRun struct {
	text                     string
	left, top, right, bottom float64
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

func groupLines(runs []textRun) []services.PDFTextLine {
	if len(runs) == 0 {
		return nil
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

	lines := make([]services.PDFTextLine, 0, len(bands))
	for _, band := range bands {
		lines = append(lines, splitBand(band)...)
	}

	return lines
}

func splitBand(band *lineBand) []services.PDFTextLine {
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

	var lines []services.PDFTextLine
	var parts []string
	var current textRun
	flush := func() {
		if len(parts) == 0 {
			return
		}
		lines = append(lines, services.PDFTextLine{
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
