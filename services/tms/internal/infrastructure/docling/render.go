package docling

import (
	"cmp"
	"math"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
)

const (
	PageHeaderMarker = "<!-- page header -->"

	maxLinesPerPage = 400
	maxLineRunes    = 160
)

type sourceProfile int

const (
	sourceOther sourceProfile = iota
	sourcePDF
	sourceImage
)

func profileOf(mimetype string) sourceProfile {
	mimetype = strings.ToLower(strings.TrimSpace(mimetype))
	switch {
	case mimetype == "application/pdf":
		return sourcePDF
	case strings.HasPrefix(mimetype, "image/"):
		return sourceImage
	default:
		return sourceOther
	}
}

type blockKind int

const (
	blockParagraph blockKind = iota
	blockList
)

type block struct {
	kind     blockKind
	markdown string
	text     string
}

type pageBuilder struct {
	number int
	size   pageSize
	header []block
	body   []block
	cells  []kernelports.ConvertedLine
	items  []kernelports.ConvertedLine
}

type renderer struct {
	doc     *doclingDocument
	pages   map[int]*pageBuilder
	visited map[string]struct{}
	current int
}

func renderPages(doc *doclingDocument, confidence *confidenceScores) []kernelports.ConvertedPage {
	r := &renderer{
		doc:     doc,
		pages:   make(map[int]*pageBuilder, max(len(doc.Pages), 1)),
		visited: make(map[string]struct{}, len(doc.Texts)+len(doc.Tables)+len(doc.Groups)),
		current: 1,
	}
	for key, value := range doc.Pages {
		number := value.PageNo
		if number <= 0 {
			number, _ = strconv.Atoi(key)
		}
		if number > 0 {
			r.page(number).size = value.Size
		}
	}

	r.walk(doc.Body.Children)
	r.walk(doc.Furniture.Children)

	if len(r.pages) == 0 {
		r.page(1)
	}

	var origin string
	if doc.Origin != nil {
		origin = doc.Origin.Mimetype
	}
	ocr, native, ocrConfidence := sourceFlags(profileOf(origin), confidence)

	numbers := make([]int, 0, len(r.pages))
	for number := range r.pages {
		numbers = append(numbers, number)
	}
	slices.Sort(numbers)

	pages := make([]kernelports.ConvertedPage, 0, len(numbers))
	for _, number := range numbers {
		builder := r.pages[number]
		pages = append(pages, kernelports.ConvertedPage{
			Number:        number,
			OCR:           ocr,
			Native:        native,
			OCRConfidence: ocrConfidence,
			Width:         builder.size.Width,
			Height:        builder.size.Height,
			Text:          builder.plainText(),
			Markdown:      builder.markdown(),
			Lines:         builder.lines(),
		})
	}

	return pages
}

func sourceFlags(
	profile sourceProfile,
	confidence *confidenceScores,
) (ocr, native bool, ocrConfidence float64) {
	if confidence != nil && confidence.OCRScore != nil {
		ocrConfidence = math.Max(0, math.Min(1, *confidence.OCRScore))
	}

	switch profile {
	case sourceImage:
		return true, false, ocrConfidence
	case sourcePDF:
		if confidence == nil {
			return false, true, 0
		}
		return confidence.OCRScore != nil, confidence.ParseScore != nil, ocrConfidence
	case sourceOther:
		return false, true, 0
	default:
		return false, true, 0
	}
}

func (r *renderer) page(number int) *pageBuilder {
	builder, ok := r.pages[number]
	if !ok {
		builder = &pageBuilder{number: number}
		r.pages[number] = builder
	}

	return builder
}

func (r *renderer) pageOf(provs []prov) int {
	for i := range provs {
		if provs[i].PageNo > 0 {
			return provs[i].PageNo
		}
	}

	return r.current
}

func (r *renderer) walk(children []ref) {
	for _, child := range children {
		if _, seen := r.visited[child.Ref]; seen {
			continue
		}
		r.visited[child.Ref] = struct{}{}

		kind, index := r.doc.resolve(child.Ref)
		switch kind {
		case itemText:
			r.text(&r.doc.Texts[index])
		case itemTable:
			r.table(&r.doc.Tables[index])
		case itemGroup:
			r.walk(r.doc.Groups[index].Children)
		case itemPicture:
			r.walk(r.doc.Pictures[index].Children)
		case itemUnknown:
		}
	}
}

func (r *renderer) text(item *textItem) {
	defer r.walk(item.Children)

	text := strings.TrimSpace(item.Text)
	if text == "" {
		return
	}

	r.current = r.pageOf(item.Prov)
	builder := r.page(r.current)
	formatted := formatText(item, text)
	if isFurniture(&item.node) {
		builder.header = append(builder.header, formatted)
	} else {
		builder.body = append(builder.body, formatted)
	}

	for i := range item.Prov {
		provenance := &item.Prov[i]
		target := builder
		if provenance.PageNo > 0 {
			target = r.page(provenance.PageNo)
		}
		if line, ok := toLine(
			spanOf(text, provenance, len(item.Prov)),
			&provenance.BBox,
			target.size,
		); ok {
			target.items = append(target.items, line)
		}
	}
}

func isFurniture(n *node) bool {
	return n.ContentLayer == contentLayerFurniture ||
		n.Label == labelPageHeader ||
		n.Label == labelPageFooter
}

func formatText(item *textItem, text string) block {
	switch item.Label {
	case labelTitle:
		return block{markdown: "# " + text, text: text}
	case labelSectionHeader:
		level := min(max(item.Level, 1)+1, 6)
		return block{markdown: strings.Repeat("#", level) + " " + text, text: text}
	case labelListItem:
		marker := "-"
		if item.Enumerated && strings.TrimSpace(item.Marker) != "" {
			marker = strings.TrimSpace(item.Marker)
		}
		return block{kind: blockList, markdown: marker + " " + text, text: text}
	case labelCheckboxSelected:
		return block{kind: blockList, markdown: "- [x] " + text, text: text}
	case labelCheckboxUnselected:
		return block{kind: blockList, markdown: "- [ ] " + text, text: text}
	case labelCode:
		return block{markdown: "```\n" + text + "\n```", text: text}
	case labelFormula:
		return block{markdown: "$$" + text + "$$", text: text}
	default:
		return block{markdown: text, text: text}
	}
}

func spanOf(text string, provenance *prov, provCount int) string {
	if provCount < 2 || len(provenance.Charspan) != 2 {
		return text
	}
	runes := []rune(text)
	start := min(max(provenance.Charspan[0], 0), len(runes))
	end := min(max(provenance.Charspan[1], start), len(runes))
	if end <= start {
		return text
	}

	return string(runes[start:end])
}

func (r *renderer) table(item *tableItem) {
	defer r.walk(item.Children)

	grid := buildGrid(&item.Data)
	if len(grid) == 0 {
		return
	}

	r.current = r.pageOf(item.Prov)
	builder := r.page(r.current)
	builder.body = append(builder.body, block{
		markdown: pipeTable(grid),
		text:     plainTable(grid),
	})

	for i := range item.Data.TableCells {
		cell := &item.Data.TableCells[i]
		if cell.BBox == nil {
			continue
		}
		if line, ok := toLine(cell.Text, cell.BBox, builder.size); ok {
			builder.cells = append(builder.cells, line)
		}
	}
}

func buildGrid(data *tableData) [][]string {
	rows, cols := data.NumRows, data.NumCols
	for i := range data.TableCells {
		cell := &data.TableCells[i]
		rows = max(rows, cell.EndRow, cell.StartRow+1)
		cols = max(cols, cell.EndCol, cell.StartCol+1)
	}
	if rows == 0 || cols == 0 {
		return nil
	}

	grid := make([][]string, rows)
	for row := range grid {
		grid[row] = make([]string, cols)
	}
	for i := range data.TableCells {
		cell := &data.TableCells[i]
		text := strings.Join(strings.Fields(cell.Text), " ")
		endRow := min(max(cell.EndRow, cell.StartRow+1), rows)
		endCol := min(max(cell.EndCol, cell.StartCol+1), cols)
		for row := max(cell.StartRow, 0); row < endRow; row++ {
			for col := max(cell.StartCol, 0); col < endCol; col++ {
				grid[row][col] = text
			}
		}
	}

	kept := grid[:0]
	for row, cells := range grid {
		if row == 0 || slices.ContainsFunc(cells, func(cell string) bool { return cell != "" }) {
			kept = append(kept, cells)
		}
	}
	if !slices.ContainsFunc(kept, func(cells []string) bool {
		return slices.ContainsFunc(cells, func(cell string) bool { return cell != "" })
	}) {
		return nil
	}

	return kept
}

func pipeTable(grid [][]string) string {
	var b strings.Builder
	writeRow := func(cells []string) {
		b.WriteString("|")
		for _, cell := range cells {
			b.WriteString(" ")
			b.WriteString(strings.ReplaceAll(cell, "|", `\|`))
			b.WriteString(" |")
		}
	}

	writeRow(grid[0])
	b.WriteString("\n|")
	for range grid[0] {
		b.WriteString(" --- |")
	}
	for _, cells := range grid[1:] {
		b.WriteString("\n")
		writeRow(cells)
	}

	return b.String()
}

func plainTable(grid [][]string) string {
	rows := make([]string, 0, len(grid))
	for _, cells := range grid {
		filled := make([]string, 0, len(cells))
		for col, cell := range cells {
			if cell == "" || (col > 0 && cell == cells[col-1]) {
				continue
			}
			filled = append(filled, cell)
		}
		if len(filled) > 0 {
			rows = append(rows, strings.Join(filled, kernelports.PDFColumnGap))
		}
	}

	return strings.Join(rows, "\n")
}

func toLine(text string, box *bbox, size pageSize) (kernelports.ConvertedLine, bool) {
	text = strings.Join(strings.Fields(text), " ")
	if text == "" || size.Width <= 0 || size.Height <= 0 {
		return kernelports.ConvertedLine{}, false
	}
	if utf8.RuneCountInString(text) > maxLineRunes {
		text = string([]rune(text)[:maxLineRunes])
	}

	left, right := math.Min(box.L, box.R), math.Max(box.L, box.R)
	top, bottom := math.Min(box.T, box.B), math.Max(box.T, box.B)
	if box.CoordOrigin == coordOriginBottomLeft {
		top, bottom = size.Height-math.Max(box.T, box.B), size.Height-math.Min(box.T, box.B)
	}

	x := fraction(left, size.Width)
	y := fraction(top, size.Height)
	w := round4(math.Min(fraction(right-left, size.Width), 1-x))
	h := round4(math.Min(fraction(bottom-top, size.Height), 1-y))
	if w <= 0 || h <= 0 {
		return kernelports.ConvertedLine{}, false
	}

	return kernelports.ConvertedLine{Text: text, X: x, Y: y, W: w, H: h}, true
}

func fraction(value, of float64) float64 {
	if of <= 0 {
		return 0
	}

	return round4(math.Max(0, math.Min(1, value/of)))
}

func round4(value float64) float64 {
	return math.Round(value*10000) / 10000
}

func (p *pageBuilder) lines() []kernelports.ConvertedLine {
	items := slices.Clone(p.items)
	slices.SortStableFunc(items, func(a, b kernelports.ConvertedLine) int {
		return cmp.Compare(utf8.RuneCountInString(a.Text), utf8.RuneCountInString(b.Text))
	})

	lines := make([]kernelports.ConvertedLine, 0, min(len(p.cells)+len(items), maxLinesPerPage))
	for _, group := range [][]kernelports.ConvertedLine{p.cells, items} {
		for _, line := range group {
			if len(lines) == maxLinesPerPage {
				return lines
			}
			lines = append(lines, line)
		}
	}

	return lines
}

func (p *pageBuilder) markdown() string {
	var b strings.Builder
	if len(p.header) > 0 {
		b.WriteString(PageHeaderMarker)
		for _, item := range p.header {
			b.WriteString("\n")
			b.WriteString(item.text)
		}
	}
	for i, item := range p.body {
		if b.Len() > 0 {
			if i > 0 && item.kind == blockList && p.body[i-1].kind == blockList {
				b.WriteString("\n")
			} else {
				b.WriteString("\n\n")
			}
		}
		b.WriteString(item.markdown)
	}

	return b.String()
}

func (p *pageBuilder) plainText() string {
	parts := make([]string, 0, len(p.header)+len(p.body))
	for _, group := range [][]block{p.header, p.body} {
		for _, item := range group {
			parts = append(parts, item.text)
		}
	}

	return strings.Join(parts, "\n")
}
