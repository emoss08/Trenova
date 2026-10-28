package agentruntime

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type cutRow struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

type cutList struct {
	Count      int      `json:"count"`
	HasMore    bool     `json:"hasMore"`
	NextCursor string   `json:"nextCursor,omitempty"`
	Items      []cutRow `json:"items"`

	cuttable bool
	asked    []int
}

func (l *cutList) PageRows() int { return len(l.Items) }

func (l *cutList) CutPage(rows int) (any, bool) {
	l.asked = append(l.asked, rows)
	if !l.cuttable {
		return nil, false
	}

	return &cutList{
		Count:      rows,
		HasMore:    true,
		NextCursor: "after-" + l.Items[rows-1].ID,
		Items:      l.Items[:rows],
		cuttable:   true,
	}, true
}

func longList(rows int, cuttable bool) *cutList {
	list := &cutList{cuttable: cuttable}
	for idx := range rows {
		list.Items = append(list.Items, cutRow{
			ID:   fmt.Sprintf("row%03d", idx),
			Text: strings.Repeat("x", 900),
		})
	}
	list.Count = rows

	return list
}

// Fifty rows of 900 characters do not fit a result: the page is cut to the
// most rows that do, and its cursor starts at the first row it left out, so
// paging on reads every row once.
func TestFitPage_CutsToTheRowsThatFitAndResumesAfterThem(t *testing.T) {
	t.Parallel()

	list := longList(50, true)
	encoded, err := encodeToolResult(list, transcriptNow, "UTC")
	require.NoError(t, err)
	require.Greater(t, len(encoded), maxToolResultBytes)

	fitted, cut := fitPage(list, encoded, transcriptNow, "UTC", nil)
	require.True(t, cut)
	assert.LessOrEqual(t, len(fitted.encoded), maxToolResultBytes)

	page, ok := fitted.data.(*cutList)
	require.True(t, ok)
	assert.Equal(t, fitted.kept, len(page.Items))
	assert.True(t, page.HasMore)
	assert.Equal(t, "after-"+page.Items[len(page.Items)-1].ID, page.NextCursor)

	longer, _ := list.CutPage(fitted.kept + 1)
	longerEncoded, err := encodeToolResult(longer, transcriptNow, "UTC")
	require.NoError(t, err)
	assert.Greater(t, len(longerEncoded), maxToolResultBytes, "one row more would not have fit")
	assert.Contains(t, fitted.note(), "no row is skipped")
	assert.Contains(t, fitted.note(), strconv.Itoa(fitted.kept)+" of the 50")
}

func TestFitPage_LeavesWhatFitsAndWhatCannotResume(t *testing.T) {
	t.Parallel()

	short := longList(3, true)
	encoded, err := encodeToolResult(short, transcriptNow, "UTC")
	require.NoError(t, err)
	_, cut := fitPage(short, encoded, transcriptNow, "UTC", nil)
	assert.False(t, cut, "a page that fits is not cut")
	assert.Empty(t, short.asked)

	ranked := longList(50, false)
	encoded, err = encodeToolResult(ranked, transcriptNow, "UTC")
	require.NoError(t, err)
	_, cut = fitPage(ranked, encoded, transcriptNow, "UTC", nil)
	assert.False(t, cut, "a page that cannot say where it goes on is left to the fence")

	plain := map[string]any{"items": longList(50, true).Items}
	encoded, err = encodeToolResult(plain, transcriptNow, "UTC")
	require.NoError(t, err)
	_, cut = fitPage(plain, encoded, transcriptNow, "UTC", nil)
	assert.False(t, cut, "a result that is not a page is left to the fence")
}

// fields narrows the model's copy of a list's rows and leaves the list whole,
// so the person's table keeps every column; what the list works out beside
// its columns stays, and a result naming no columns is left as it is.
func TestNarrowRows_NarrowsOnlyWhatTheModelReads(t *testing.T) {
	t.Parallel()

	list := map[string]any{
		"columns": []string{"id", "wholeName", "city", "profile"},
		"items": []map[string]any{{
			"id": "wrk_1", "wholeName": "Ann", "city": "Dallas",
			"profile":      map[string]any{"cdlClass": "A"},
			"dispatchable": true,
		}},
	}
	encoded, _, err := encodeModelDocument(list, transcriptNow, "UTC", []string{"wholeName"})
	require.NoError(t, err)

	assert.Contains(t, encoded, `"wholeName":"Ann"`)
	assert.Contains(t, encoded, `"id":"wrk_1"`)
	assert.Contains(t, encoded, `"dispatchable":true`, "what the list works out is kept")
	assert.NotContains(t, encoded, "Dallas")
	assert.NotContains(t, encoded, "cdlClass")
	assert.Contains(t, encoded, `"columns":["id","wholeName"]`)
	assert.Equal(t, "Dallas", list["items"].([]map[string]any)[0]["city"],
		"the list itself, which the table is drawn from, keeps every column")

	unnamed := map[string]any{"items": []map[string]any{{"id": "a", "city": "Waco"}}}
	encoded, _, err = encodeModelDocument(unnamed, transcriptNow, "UTC", []string{"wholeName"})
	require.NoError(t, err)
	assert.Contains(t, encoded, "Waco", "a result naming no columns is left whole")

	assert.Equal(t, []string{"wholeName", "city"}, listFields(map[string]any{
		"fields": []any{"wholeName", "", "city"},
	}))
}
