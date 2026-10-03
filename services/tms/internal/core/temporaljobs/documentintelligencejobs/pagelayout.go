package documentintelligencejobs

import (
	"html"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/core/domain/documentcontent"
)

// The layout of a page is kept beside its text so a value the extraction
// read can be pointed at on the page: a native PDF says where each line of
// text sits, and the OCR engine where each word it recognised was.

const (
	maxLayoutLineRunes = 160
	// glyphWidthEm is how wide an average glyph is against its font size.
	// The PDF's text layer gives a line's start but not its end, so a line's
	// width is estimated from its length.
	glyphWidthEm = 0.5
)

type tsvLineKey struct {
	page, block, par, line int
}

type layoutBox struct {
	text                   []string
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
		lines = append(lines, layoutLine(
			strings.Join(box.text, " "),
			box.left, box.top, box.right-box.left, box.bott-box.top,
			extentW, extentH,
		))
	}

	return lines
}

var (
	fitzPageStyle = regexp.MustCompile(`<div[^>]*style="[^"]*width:([0-9.]+)pt;height:([0-9.]+)pt`)
	fitzLine      = regexp.MustCompile(`(?s)<p style="([^"]*)">(.*?)</p>`)
	fitzStyleNum  = regexp.MustCompile(`(top|left|line-height|font-size):([0-9.]+)pt`)
	fitzTag       = regexp.MustCompile(`<[^>]+>`)
)

// fitzLayout reads the lines of a native PDF page out of MuPDF's structured
// text, which gives each line's top, left and height in points.
func fitzLayout(page string) []documentcontent.LayoutLine {
	size := fitzPageStyle.FindStringSubmatch(page)
	if size == nil {
		return nil
	}
	pageW, _ := strconv.ParseFloat(size[1], 64)
	pageH, _ := strconv.ParseFloat(size[2], 64)
	if pageW <= 0 || pageH <= 0 {
		return nil
	}

	var lines []documentcontent.LayoutLine
	for _, match := range fitzLine.FindAllStringSubmatch(page, -1) {
		if len(lines) == documentcontent.MaxLayoutLines {
			break
		}
		text := strings.Join(strings.Fields(html.UnescapeString(fitzTag.ReplaceAllString(match[2], " "))), " ")
		if text == "" {
			continue
		}
		style := map[string]float64{}
		for _, part := range fitzStyleNum.FindAllStringSubmatch(match[1]+" "+match[2], -1) {
			if _, set := style[part[1]]; set {
				continue
			}
			style[part[1]], _ = strconv.ParseFloat(part[2], 64)
		}
		height := style["line-height"]
		if height <= 0 {
			height = style["font-size"]
		}
		fontSize := style["font-size"]
		if fontSize <= 0 {
			fontSize = height
		}
		width := float64(utf8.RuneCountInString(text)) * fontSize * glyphWidthEm
		lines = append(lines, layoutLine(text, style["left"], style["top"], width, height, pageW, pageH))
	}

	return lines
}

func layoutLine(text string, left, top, width, height, pageW, pageH float64) documentcontent.LayoutLine {
	if utf8.RuneCountInString(text) > maxLayoutLineRunes {
		text = string([]rune(text)[:maxLayoutLineRunes])
	}
	x := documentcontent.Fraction(left, pageW)
	y := documentcontent.Fraction(top, pageH)

	return documentcontent.LayoutLine{
		Text: text,
		X:    x,
		Y:    y,
		W:    min(documentcontent.Fraction(width, pageW), 1-x),
		H:    min(documentcontent.Fraction(height, pageH), 1-y),
	}
}

// pageHTML is the slice of a PDF reader that gives a page's text with where
// each line sits.
type pageHTML interface {
	HTML(pageNumber int, header bool) (string, error)
}
