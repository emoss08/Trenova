package pdftest

import (
	"bytes"
	"cmp"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/ledongthuc/pdf"
	"github.com/stretchr/testify/require"
)

type Size struct {
	Width  float64
	Height float64
}

func open(t testing.TB, data []byte) *pdf.Reader {
	t.Helper()

	reader, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	require.NoError(t, err, "the PDF could not be opened")
	return reader
}

func PageTexts(t testing.TB, data []byte) []string {
	t.Helper()

	reader := open(t, data)
	texts := make([]string, 0, reader.NumPage())
	for page := 1; page <= reader.NumPage(); page++ {
		texts = append(texts, pageText(t, reader.Page(page), page))
	}

	return texts
}

func Text(t testing.TB, data []byte) string {
	t.Helper()
	return strings.Join(PageTexts(t, data), "\n")
}

func PageCount(t testing.TB, data []byte) int {
	t.Helper()
	return open(t, data).NumPage()
}

func PageSize(t testing.TB, data []byte, page int) Size {
	t.Helper()

	reader := open(t, data)
	require.True(t, page >= 1 && page <= reader.NumPage(), "the PDF has no page %d", page)
	for node := reader.Page(page).V; !node.IsNull(); node = node.Key("Parent") {
		box := node.Key("MediaBox")
		if box.Len() != 4 {
			continue
		}
		size := Size{
			Width:  box.Index(2).Float64() - box.Index(0).Float64(),
			Height: box.Index(3).Float64() - box.Index(1).Float64(),
		}
		if rotate := node.Key("Rotate").Int64(); rotate%180 != 0 {
			size.Width, size.Height = size.Height, size.Width
		}
		return size
	}
	t.Fatalf("page %d has no media box", page)
	return Size{}
}

func pageText(t testing.TB, page pdf.Page, number int) (text string) {
	t.Helper()

	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("read the text of page %d: %v", number, fmt.Sprint(recovered))
		}
	}()
	rows, err := page.GetTextByRow()
	require.NoError(t, err, "read the text of page %d", number)
	slices.SortStableFunc(rows, func(a, b *pdf.Row) int {
		return cmp.Compare(b.Position, a.Position)
	})

	var out strings.Builder
	for _, row := range rows {
		runs := slices.Clone(row.Content)
		slices.SortStableFunc(runs, func(a, b pdf.Text) int { return cmp.Compare(a.X, b.X) })
		line := make([]string, 0, len(runs))
		var cell strings.Builder
		for i, run := range runs {
			if i > 0 && run.X != runs[i-1].X {
				line = append(line, strings.TrimSpace(cell.String()))
				cell.Reset()
			}
			cell.WriteString(run.S)
		}
		line = append(line, strings.TrimSpace(cell.String()))
		if joined := strings.TrimSpace(strings.Join(line, " ")); joined != "" {
			out.WriteString(joined)
			out.WriteByte('\n')
		}
	}
	return out.String()
}
