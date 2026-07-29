package shipment_test

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func textNode(text string, marks ...map[string]any) map[string]any {
	node := map[string]any{"type": "text", "text": text}
	if len(marks) > 0 {
		rawMarks := make([]any, 0, len(marks))
		for _, mark := range marks {
			rawMarks = append(rawMarks, mark)
		}
		node["marks"] = rawMarks
	}
	return node
}

func paragraph(children ...map[string]any) map[string]any {
	content := make([]any, 0, len(children))
	for _, child := range children {
		content = append(content, child)
	}
	return map[string]any{"type": "paragraph", "content": content}
}

func doc(blocks ...map[string]any) map[string]any {
	content := make([]any, 0, len(blocks))
	for _, block := range blocks {
		content = append(content, block)
	}
	return map[string]any{"type": "doc", "content": content}
}

func mentionNode(id pulid.ID, label string) map[string]any {
	return map[string]any{
		"type":  "mention",
		"attrs": map[string]any{"id": id.String(), "label": label},
	}
}

func entityRefNode(entityType string, id pulid.ID, label string) map[string]any {
	return map[string]any{
		"type": "entityRef",
		"attrs": map[string]any{
			"entityType": entityType,
			"entityId":   id.String(),
			"label":      label,
		},
	}
}

func validateBody(body map[string]any) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	shipment.ValidateCommentBody(body, multiErr)
	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

func TestValidateCommentBody_AcceptsValidDocument(t *testing.T) {
	t.Parallel()

	userID := pulid.MustNew("usr_")
	shipmentID := pulid.MustNew("shp_")
	body := doc(
		paragraph(
			textNode("Hello ", map[string]any{"type": "bold"}),
			mentionNode(userID, "Jane Doe"),
			textNode(" check "),
			entityRefNode("shipment", shipmentID, "S-100045"),
		),
		map[string]any{
			"type": "bulletList",
			"content": []any{
				map[string]any{
					"type":    "listItem",
					"content": []any{paragraph(textNode("first"))},
				},
				map[string]any{
					"type":    "listItem",
					"content": []any{paragraph(textNode("second"))},
				},
			},
		},
	)

	require.Nil(t, validateBody(body))
}

func TestValidateCommentBody_NilBodyIsAccepted(t *testing.T) {
	t.Parallel()

	require.Nil(t, validateBody(nil))
}

func TestValidateCommentBody_RejectsNonDocRoot(t *testing.T) {
	t.Parallel()

	require.NotNil(t, validateBody(map[string]any{"type": "paragraph"}))
}

func TestValidateCommentBody_RejectsEmptyDocument(t *testing.T) {
	t.Parallel()

	require.NotNil(t, validateBody(doc()))
	require.NotNil(t, validateBody(doc(paragraph(textNode("   ")))))
}

func TestValidateCommentBody_RejectsUnknownNodesAndMarks(t *testing.T) {
	t.Parallel()

	require.NotNil(t, validateBody(doc(map[string]any{"type": "heading"})))
	require.NotNil(t, validateBody(doc(paragraph(map[string]any{"type": "image"}))))
	require.NotNil(t, validateBody(doc(paragraph(
		textNode("styled", map[string]any{"type": "textStyle"}),
	))))
}

func TestValidateCommentBody_RejectsUnsafeLinks(t *testing.T) {
	t.Parallel()

	for _, href := range []string{
		"javascript:alert(1)",
		"data:text/html;base64,PHNjcmlwdD4=",
		"ftp://example.com/file",
		"",
	} {
		body := doc(paragraph(textNode("link", map[string]any{
			"type":  "link",
			"attrs": map[string]any{"href": href},
		})))
		require.NotNil(t, validateBody(body), "href %q must be rejected", href)
	}

	valid := doc(paragraph(textNode("link", map[string]any{
		"type":  "link",
		"attrs": map[string]any{"href": "https://example.com/x"},
	})))
	require.Nil(t, validateBody(valid))
}

func TestValidateCommentBody_RejectsInvalidMentionAndEntityRef(t *testing.T) {
	t.Parallel()

	require.NotNil(t, validateBody(doc(paragraph(map[string]any{
		"type":  "mention",
		"attrs": map[string]any{"id": "not-a-pulid", "label": "Jane"},
	}))))
	require.NotNil(t, validateBody(doc(paragraph(map[string]any{
		"type":  "mention",
		"attrs": map[string]any{"id": pulid.MustNew("usr_").String(), "label": ""},
	}))))
	require.NotNil(t, validateBody(doc(paragraph(map[string]any{
		"type": "entityRef",
		"attrs": map[string]any{
			"entityType": "invoice",
			"entityId":   pulid.MustNew("shp_").String(),
			"label":      "X",
		},
	}))))
}

func TestValidateCommentBody_RejectsExcessiveDepth(t *testing.T) {
	t.Parallel()

	innermost := map[string]any{
		"type":    "listItem",
		"content": []any{paragraph(textNode("deep"))},
	}
	list := map[string]any{"type": "bulletList", "content": []any{innermost}}
	for range 6 {
		item := map[string]any{"type": "listItem", "content": []any{list}}
		list = map[string]any{"type": "bulletList", "content": []any{item}}
	}

	require.NotNil(t, validateBody(doc(list)))
}

func TestValidateCommentBody_RejectsOverlongText(t *testing.T) {
	t.Parallel()

	body := doc(paragraph(textNode(strings.Repeat("a", shipment.MaxCommentLength+1))))
	require.NotNil(t, validateBody(body))
}

func TestRenderCommentBodyText(t *testing.T) {
	t.Parallel()

	userID := pulid.MustNew("usr_")
	shipmentID := pulid.MustNew("shp_")
	body := doc(
		paragraph(
			textNode("Hello "),
			mentionNode(userID, "Jane"),
			textNode(" see "),
			entityRefNode("shipment", shipmentID, "S-1"),
		),
		paragraph(textNode("line one"), map[string]any{"type": "hardBreak"}, textNode("line two")),
		map[string]any{
			"type":  "orderedList",
			"attrs": map[string]any{"start": float64(3)},
			"content": []any{
				map[string]any{
					"type":    "listItem",
					"content": []any{paragraph(textNode("third"))},
				},
				map[string]any{
					"type":    "listItem",
					"content": []any{paragraph(textNode("fourth"))},
				},
			},
		},
	)

	text := shipment.RenderCommentBodyText(body)
	assert.Equal(
		t,
		"Hello @Jane see #S-1\nline one\nline two\n3. third\n4. fourth",
		text,
	)
}

func TestRenderCommentBodyText_HandlesMalformedInput(t *testing.T) {
	t.Parallel()

	assert.Empty(t, shipment.RenderCommentBodyText(nil))
	assert.Empty(t, shipment.RenderCommentBodyText(map[string]any{"type": "garbage"}))
	assert.Empty(t, shipment.RenderCommentBodyText(map[string]any{
		"type":    "doc",
		"content": []any{"not-a-node", 42, nil},
	}))
}

func TestExtractMentionUserIDs_DedupesAndSkipsInvalid(t *testing.T) {
	t.Parallel()

	userA := pulid.MustNew("usr_")
	userB := pulid.MustNew("usr_")
	body := doc(
		paragraph(
			mentionNode(userA, "A"),
			mentionNode(userB, "B"),
			mentionNode(userA, "A again"),
			map[string]any{"type": "mention", "attrs": map[string]any{"id": "bogus"}},
		),
	)

	ids := shipment.ExtractMentionUserIDs(body)
	assert.Equal(t, []pulid.ID{userA, userB}, ids)
}

func TestExtractEntityRefs_DedupesByTypeAndID(t *testing.T) {
	t.Parallel()

	shipmentID := pulid.MustNew("shp_")
	workerID := pulid.MustNew("wrk_")
	body := doc(
		paragraph(
			entityRefNode("shipment", shipmentID, "S-1"),
			entityRefNode("worker", workerID, "John"),
			entityRefNode("shipment", shipmentID, "S-1 dup"),
			entityRefNode("invoice", pulid.MustNew("inv_"), "bad type"),
		),
	)

	refs := shipment.ExtractEntityRefs(body)
	require.Len(t, refs, 2)
	assert.Equal(t, shipment.CommentEntityRefShipment, refs[0].EntityType)
	assert.Equal(t, shipmentID, refs[0].EntityID)
	assert.Equal(t, shipment.CommentEntityRefWorker, refs[1].EntityType)
	assert.Equal(t, workerID, refs[1].EntityID)
}
