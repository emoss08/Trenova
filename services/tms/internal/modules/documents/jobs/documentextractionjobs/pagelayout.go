package documentextractionjobs

import (
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/modules/documents/domain/documentcontent"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
)

// The layout of a page is kept beside its text so a value the extraction
// read can be pointed at on the page: a native PDF says where each line of
// text sits, and the OCR engine where each word it recognised was.

const maxLayoutLineRunes = 160

type tsvLineKey struct {
	page, block, par, line int
}

type layoutBox struct {
	text                   []string
	words                  []kernelports.PDFTextWord
	left, top, right, bott float64
}

func (b *layoutBox) add(word string, left, top, width, height float64) {
	if len(b.text) == 0 {
		b.left, b.top, b.right, b.bott = left, top, left+width, top+height
	} else {
		b.left = min(b.left, left)
		b.top = min(b.top, top)
		b.right = max(b.right, left+width)
		b.bott = max(b.bott, top+height)
	}
	b.text = append(b.text, word)
	b.words = append(b.words, kernelports.PDFTextWord{Left: left, Width: width})
}

// tesseractLayout groups the words of tesseract's TSV output into lines and
// places each as a fraction of the image. Width and height are the image's;
// when they are unknown the extent of the words stands in for the page.
func tesseractLayout(output string, width, height int) []documentcontent.LayoutLine {
	var order []tsvLineKey
	boxes := map[tsvLineKey]*layoutBox{}
	extentW, extentH := float64(width), float64(height)
	measured := width <= 0 || height <= 0

	for _, raw := range strings.Split(strings.ReplaceAll(output, "\r\n", "\n"), "\n") {
		cols := strings.Split(raw, "\t")
		if len(cols) < 12 {
			continue
		}
		level, err := strconv.Atoi(cols[0])
		if err != nil || level != 5 {
			continue
		}
		word := strings.TrimSpace(cols[11])
		if word == "" {
			continue
		}
		numbers := make([]float64, 0, 8)
		for _, col := range cols[1:9] {
			n, convErr := strconv.ParseFloat(strings.TrimSpace(col), 64)
			if convErr != nil {
				break
			}
			numbers = append(numbers, n)
		}
		if len(numbers) != 8 {
			continue
		}
		key := tsvLineKey{int(numbers[0]), int(numbers[1]), int(numbers[2]), int(numbers[3])}
		box, seen := boxes[key]
		if !seen {
			box = &layoutBox{}
			boxes[key] = box
			order = append(order, key)
		}
		left, top, w, h := numbers[5], numbers[6], numbers[7], 0.0
		if n, convErr := strconv.ParseFloat(strings.TrimSpace(cols[9]), 64); convErr == nil {
			h = n
		}
		box.add(word, left, top, w, h)
		if measured {
			extentW = max(extentW, left+w)
			extentH = max(extentH, top+h)
		}
	}

	lines := make([]documentcontent.LayoutLine, 0, min(len(order), documentcontent.MaxLayoutLines))
	for _, key := range order {
		if len(lines) == documentcontent.MaxLayoutLines {
			break
		}
		box := boxes[key]
		lines = append(lines, layoutLine(layoutLineParams{
			text:  strings.Join(box.text, " "),
			words: box.words,
			box: kernelports.PDFTextLine{
				Left:   box.left,
				Top:    box.top,
				Width:  box.right - box.left,
				Height: box.bott - box.top,
			},
			pageW: extentW,
			pageH: extentH,
		}))
	}

	return lines
}

func pdfLayout(layout *kernelports.PDFPageLayout) []documentcontent.LayoutLine {
	if layout == nil || layout.Width <= 0 || layout.Height <= 0 {
		return nil
	}

	lines := make(
		[]documentcontent.LayoutLine,
		0,
		min(len(layout.Lines), documentcontent.MaxLayoutLines),
	)
	for _, line := range layout.Lines {
		if len(lines) == documentcontent.MaxLayoutLines {
			break
		}
		text := strings.Join(strings.Fields(line.Text), " ")
		if text == "" {
			continue
		}
		lines = append(lines, layoutLine(layoutLineParams{
			text:  text,
			words: line.Words,
			box:   line,
			pageW: layout.Width,
			pageH: layout.Height,
		}))
	}

	return lines
}

type layoutLineParams struct {
	text  string
	words []kernelports.PDFTextWord
	box   kernelports.PDFTextLine
	pageW float64
	pageH float64
}

func layoutLine(p layoutLineParams) documentcontent.LayoutLine {
	text := p.text
	if utf8.RuneCountInString(text) > maxLayoutLineRunes {
		text = string([]rune(text)[:maxLayoutLineRunes])
	}
	x := documentcontent.Fraction(p.box.Left, p.pageW)
	y := documentcontent.Fraction(p.box.Top, p.pageH)

	return documentcontent.LayoutLine{
		Text:  text,
		X:     x,
		Y:     y,
		W:     min(documentcontent.Fraction(p.box.Width, p.pageW), 1-x),
		H:     min(documentcontent.Fraction(p.box.Height, p.pageH), 1-y),
		Words: wordSpans(text, p.words, p.pageW),
	}
}

func wordSpans(text string, words []kernelports.PDFTextWord, pageW float64) []float64 {
	count := len(strings.Fields(text))
	if count == 0 || len(words) < count || pageW <= 0 {
		return nil
	}

	spans := make([]float64, 0, count*2)
	for _, word := range words[:count] {
		start := documentcontent.Fraction(word.Left, pageW)
		spans = append(
			spans,
			start,
			max(start, documentcontent.Fraction(word.Left+word.Width, pageW)),
		)
	}

	return spans
}
