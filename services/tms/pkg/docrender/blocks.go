// Package docrender turns a markdown write-up into the files it downloads
// as: a printable HTML page for a PDF, and a Word document. It reads the
// markdown the Desk writes (headings, paragraphs, lists, pipe tables, block
// quotes, emphasis, code, links and [^N] citation marks) and nothing more;
// anything else is kept as plain text.
package docrender

import (
	"regexp"
	"strings"
)

// BlockKind is what a block of a document is.
type BlockKind int

const (
	BlockParagraph BlockKind = iota
	BlockHeading
	BlockList
	BlockTable
	BlockQuote
	BlockRule
)

// Block is one top-level part of a document.
type Block struct {
	Kind    BlockKind
	Level   int
	Text    string
	Ordered bool
	Items   []string
	Header  []string
	Rows    [][]string
}

var (
	headingLine = regexp.MustCompile(`^(#{1,6})\s+(.*)$`)
	orderedItem = regexp.MustCompile(`^\s*\d+[.)]\s+(.*)$`)
	bulletItem  = regexp.MustCompile(`^\s*[-*+]\s+(.*)$`)
	ruleLine    = regexp.MustCompile(`^\s*([-*_])(\s*[-*_]){2,}\s*$`)
	tableRule   = regexp.MustCompile(`^\s*\|?\s*:?-{2,}:?\s*(\|\s*:?-{2,}:?\s*)*\|?\s*$`)
)

// Parse splits markdown into its top-level blocks.
func Parse(markdown string) []Block {
	lines := strings.Split(strings.ReplaceAll(markdown, "\r\n", "\n"), "\n")
	var blocks []Block
	var paragraph []string
	flush := func() {
		if len(paragraph) > 0 {
			blocks = append(blocks, Block{
				Kind: BlockParagraph,
				Text: strings.Join(paragraph, " "),
			})
			paragraph = nil
		}
	}

	for idx := 0; idx < len(lines); idx++ {
		line := strings.TrimRight(lines[idx], " \t")
		trimmed := strings.TrimSpace(line)
		switch {
		case trimmed == "":
			flush()
		case headingLine.MatchString(trimmed):
			flush()
			match := headingLine.FindStringSubmatch(trimmed)
			blocks = append(blocks, Block{
				Kind:  BlockHeading,
				Level: len(match[1]),
				Text:  strings.TrimSpace(strings.TrimRight(match[2], "#")),
			})
		case ruleLine.MatchString(trimmed) && len(paragraph) == 0:
			blocks = append(blocks, Block{Kind: BlockRule})
		case orderedItem.MatchString(line) || bulletItem.MatchString(line):
			flush()
			ordered := orderedItem.MatchString(line)
			block := Block{Kind: BlockList, Ordered: ordered}
			for ; idx < len(lines); idx++ {
				item := lines[idx]
				switch {
				case orderedItem.MatchString(item) && ordered:
					block.Items = append(block.Items, orderedItem.FindStringSubmatch(item)[1])
				case bulletItem.MatchString(item) && !ordered:
					block.Items = append(block.Items, bulletItem.FindStringSubmatch(item)[1])
				case strings.TrimSpace(item) != "" && len(block.Items) > 0 &&
					strings.HasPrefix(item, "  "):
					block.Items[len(block.Items)-1] += " " + strings.TrimSpace(item)
				default:
					idx--
					goto listDone
				}
			}
		listDone:
			blocks = append(blocks, block)
		case strings.HasPrefix(trimmed, "|") && idx+1 < len(lines) && tableRule.MatchString(lines[idx+1]):
			flush()
			block := Block{Kind: BlockTable, Header: tableCells(trimmed)}
			idx += 2
			for ; idx < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[idx]), "|"); idx++ {
				block.Rows = append(block.Rows, tableCells(strings.TrimSpace(lines[idx])))
			}
			idx--
			blocks = append(blocks, block)
		case strings.HasPrefix(trimmed, ">"):
			flush()
			var quoted []string
			for ; idx < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[idx]), ">"); idx++ {
				quoted = append(quoted, strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(lines[idx]), ">")))
			}
			idx--
			blocks = append(blocks, Block{Kind: BlockQuote, Text: strings.Join(quoted, " ")})
		default:
			paragraph = append(paragraph, trimmed)
		}
	}
	flush()

	return blocks
}

func tableCells(line string) []string {
	line = strings.TrimSuffix(strings.TrimPrefix(line, "|"), "|")
	parts := strings.Split(line, "|")
	cells := make([]string, 0, len(parts))
	for _, part := range parts {
		cells = append(cells, strings.TrimSpace(part))
	}

	return cells
}

// SpanKind is how a run of inline text is set.
type SpanKind int

const (
	SpanText SpanKind = iota
	SpanBold
	SpanItalic
	SpanCode
	SpanLink
	SpanCite
)

// Span is one run of inline text.
type Span struct {
	Kind SpanKind
	Text string
	Href string
}

var inline = regexp.MustCompile(
	"\\[\\^(\\d{1,3})\\]|\\*\\*([^*]+)\\*\\*|__([^_]+)__|`([^`]+)`|\\[([^\\]]+)\\]\\(([^)\\s]+)\\)|\\*([^*]+)\\*|_([^_]+)_",
)

// Inline splits a block's text into its runs.
func Inline(text string) []Span {
	var spans []Span
	last := 0
	for _, match := range inline.FindAllStringSubmatchIndex(text, -1) {
		if match[0] > last {
			spans = append(spans, Span{Kind: SpanText, Text: text[last:match[0]]})
		}
		group := func(n int) string {
			if match[2*n] < 0 {
				return ""
			}
			return text[match[2*n]:match[2*n+1]]
		}
		switch {
		case group(1) != "":
			spans = append(spans, Span{Kind: SpanCite, Text: group(1)})
		case group(2) != "":
			spans = append(spans, Span{Kind: SpanBold, Text: group(2)})
		case group(3) != "":
			spans = append(spans, Span{Kind: SpanBold, Text: group(3)})
		case group(4) != "":
			spans = append(spans, Span{Kind: SpanCode, Text: group(4)})
		case group(5) != "":
			spans = append(spans, Span{Kind: SpanLink, Text: group(5), Href: group(6)})
		case group(7) != "":
			spans = append(spans, Span{Kind: SpanItalic, Text: group(7)})
		case group(8) != "":
			spans = append(spans, Span{Kind: SpanItalic, Text: group(8)})
		}
		last = match[1]
	}
	if last < len(text) {
		spans = append(spans, Span{Kind: SpanText, Text: text[last:]})
	}

	return spans
}

// PlainText is a block's text with its markup taken out and its citation
// marks kept as [N].
func PlainText(text string) string {
	var b strings.Builder
	for _, span := range Inline(text) {
		if span.Kind == SpanCite {
			b.WriteString("[" + span.Text + "]")
			continue
		}
		b.WriteString(span.Text)
	}

	return b.String()
}
