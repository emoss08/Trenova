package agentruntime

import (
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

// fittedPage is a list page cut to what the model has room to read: the page
// as cut, its encoding for the model, and how many of the page's rows it kept.
type fittedPage struct {
	data     any
	encoded  string
	document any
	kept     int
	read     int
}

// fitPage cuts a list page whose result is too long for the model to read
// whole to the most rows that fit, so the page and its cursor agree. The fence
// would otherwise shorten the encoded list and keep the page's own cursor,
// and every row between the cut and the page's end would be skipped by the
// next page. A page that cannot say where a shorter page goes on, or whose
// first row alone does not fit, is left to the fence.
//
// Only a model's read is cut: a script reads the whole page into its own
// limits, and a person's table pages with its own size.
func fitPage(
	data any,
	encoded string,
	now int64,
	timezone string,
	fields []string,
) (fittedPage, bool) {
	if len(encoded) <= maxToolResultBytes {
		return fittedPage{}, false
	}
	page, ok := data.(agenttoolports.PageCutter)
	if !ok {
		return fittedPage{}, false
	}
	rows := page.PageRows()
	if rows < 2 {
		return fittedPage{}, false
	}

	fit := func(kept int) (fittedPage, bool) {
		cut, cuttable := page.CutPage(kept)
		if !cuttable {
			return fittedPage{}, false
		}
		text, document, err := encodeModelDocument(cut, now, timezone, fields)
		if err != nil || len(text) > maxToolResultBytes {
			return fittedPage{}, false
		}

		return fittedPage{data: cut, encoded: text, document: document, kept: kept, read: rows},
			true
	}

	best, found := fit(1)
	if !found {
		return fittedPage{}, false
	}
	low, high := 2, rows-1
	for low <= high {
		mid := low + (high-low)/2
		if candidate, fits := fit(mid); fits {
			best = candidate
			low = mid + 1
			continue
		}
		high = mid - 1
	}

	return best, true
}

// note tells the model the page was cut and that nothing was skipped, and
// steers a question about the whole list away from reading it page by page.
func (p *fittedPage) note() string {
	return fmt.Sprintf("[This page was cut to its first %d of the %d rows read, as many as "+
		"fit in one result, and its nextCursor starts at the row after them, so no row is "+
		"skipped. The person's table pages on from there. Read the next page only when the "+
		"answer needs rows past these. For a question about the whole list (how many, "+
		"which ones lack something, every one of them), narrow it with the tool's own "+
		"filters, name in fields only the columns the answer needs where the tool takes "+
		"fields, or, when you hold analysis_run, read every page in one script, rather "+
		"than paging through it here.]",
		p.kept, p.read)
}
