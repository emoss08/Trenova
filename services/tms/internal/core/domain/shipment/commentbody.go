package shipment

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	CommentBodyMaxDepth = 6
	CommentBodyMaxNodes = 5000
	CommentBodyMaxLink  = 2048
)

const (
	commentNodeDoc         = "doc"
	commentNodeParagraph   = "paragraph"
	commentNodeBulletList  = "bulletList"
	commentNodeOrderedList = "orderedList"
	commentNodeListItem    = "listItem"
	commentNodeText        = "text"
	commentNodeMention     = "mention"
	commentNodeEntityRef   = "entityRef"
	commentNodeHardBreak   = "hardBreak"
)

const (
	commentMarkBold      = "bold"
	commentMarkItalic    = "italic"
	commentMarkUnderline = "underline"
	commentMarkStrike    = "strike"
	commentMarkLink      = "link"
)

type CommentEntityRefType string

const (
	CommentEntityRefShipment CommentEntityRefType = "shipment"
	CommentEntityRefWorker   CommentEntityRefType = "worker"
	CommentEntityRefCustomer CommentEntityRefType = "customer"
)

type CommentEntityRef struct {
	EntityType CommentEntityRefType `json:"entityType"`
	EntityID   pulid.ID             `json:"entityId"`
	Label      string               `json:"label"`
}

var commentEntityRefTypes = map[string]bool{
	string(CommentEntityRefShipment): true,
	string(CommentEntityRefWorker):   true,
	string(CommentEntityRefCustomer): true,
}

var commentInlineMarks = map[string]bool{
	commentMarkBold:      true,
	commentMarkItalic:    true,
	commentMarkUnderline: true,
	commentMarkStrike:    true,
	commentMarkLink:      true,
}

type commentBodyWalker struct {
	multiErr  *errortypes.MultiError
	nodeCount int
}

func ValidateCommentBody(body map[string]any, multiErr *errortypes.MultiError) {
	if body == nil {
		return
	}

	w := &commentBodyWalker{multiErr: multiErr}
	if nodeType(body) != commentNodeDoc {
		multiErr.Add("body", errortypes.ErrInvalid, "Body root must be a doc node")
		return
	}

	content := nodeContent(body)
	if len(content) == 0 {
		multiErr.Add("body", errortypes.ErrInvalid, "Body must contain at least one block")
		return
	}

	for i, block := range content {
		w.validateBlock(block, 1, fmt.Sprintf("body.content[%d]", i))
	}

	text := RenderCommentBodyText(body)
	if strings.TrimSpace(text) == "" {
		multiErr.Add("body", errortypes.ErrInvalid, "Body must contain text content")
	}
	if len(text) > MaxCommentLength {
		multiErr.Add(
			"body",
			errortypes.ErrInvalid,
			"Body text must be 5000 characters or fewer",
		)
	}
}

func (w *commentBodyWalker) countNode(path string) bool {
	w.nodeCount++
	if w.nodeCount > CommentBodyMaxNodes {
		w.multiErr.Add(path, errortypes.ErrInvalid, "Body exceeds the maximum node count")
		return false
	}
	return true
}

func (w *commentBodyWalker) validateBlock(node map[string]any, depth int, path string) {
	if node == nil {
		w.multiErr.Add(path, errortypes.ErrInvalid, "Block node must be an object")
		return
	}
	if !w.countNode(path) {
		return
	}
	if depth > CommentBodyMaxDepth {
		w.multiErr.Add(path, errortypes.ErrInvalid, "Body exceeds the maximum nesting depth")
		return
	}

	switch nodeType(node) {
	case commentNodeParagraph:
		for i, inline := range nodeContent(node) {
			w.validateInline(inline, fmt.Sprintf("%s.content[%d]", path, i))
		}
	case commentNodeBulletList, commentNodeOrderedList:
		content := nodeContent(node)
		if len(content) == 0 {
			w.multiErr.Add(path, errortypes.ErrInvalid, "List must contain at least one item")
			return
		}
		for i, item := range content {
			itemPath := fmt.Sprintf("%s.content[%d]", path, i)
			if nodeType(item) != commentNodeListItem {
				w.multiErr.Add(itemPath, errortypes.ErrInvalid, "List content must be list items")
				continue
			}
			if !w.countNode(itemPath) {
				return
			}
			for j, child := range nodeContent(item) {
				childPath := fmt.Sprintf("%s.content[%d]", itemPath, j)
				switch nodeType(child) {
				case commentNodeParagraph, commentNodeBulletList, commentNodeOrderedList:
					w.validateBlock(child, depth+2, childPath)
				default:
					w.multiErr.Add(
						childPath,
						errortypes.ErrInvalid,
						"List items may only contain paragraphs and nested lists",
					)
				}
			}
		}
	default:
		w.multiErr.Add(
			path,
			errortypes.ErrInvalid,
			"Block type must be paragraph, bulletList, or orderedList",
		)
	}
}

func (w *commentBodyWalker) validateInline(node map[string]any, path string) {
	if node == nil {
		w.multiErr.Add(path, errortypes.ErrInvalid, "Inline node must be an object")
		return
	}
	if !w.countNode(path) {
		return
	}

	switch nodeType(node) {
	case commentNodeText:
		text, _ := node["text"].(string)
		if text == "" {
			w.multiErr.Add(path, errortypes.ErrInvalid, "Text node must contain text")
		}
		w.validateMarks(node, path)
	case commentNodeMention:
		attrs := nodeAttrs(node)
		id, _ := attrs["id"].(string)
		if _, err := pulid.Parse(id); err != nil {
			w.multiErr.Add(path, errortypes.ErrInvalid, "Mention must reference a valid user ID")
		}
		if label, _ := attrs["label"].(string); label == "" {
			w.multiErr.Add(path, errortypes.ErrInvalid, "Mention must carry a label")
		}
	case commentNodeEntityRef:
		attrs := nodeAttrs(node)
		entityType, _ := attrs["entityType"].(string)
		if !commentEntityRefTypes[entityType] {
			w.multiErr.Add(
				path,
				errortypes.ErrInvalid,
				"Entity reference type must be shipment, worker, or customer",
			)
		}
		entityID, _ := attrs["entityId"].(string)
		if _, err := pulid.Parse(entityID); err != nil {
			w.multiErr.Add(
				path,
				errortypes.ErrInvalid,
				"Entity reference must carry a valid entity ID",
			)
		}
		if label, _ := attrs["label"].(string); label == "" {
			w.multiErr.Add(path, errortypes.ErrInvalid, "Entity reference must carry a label")
		}
	case commentNodeHardBreak:
	default:
		w.multiErr.Add(
			path,
			errortypes.ErrInvalid,
			"Inline type must be text, mention, entityRef, or hardBreak",
		)
	}
}

func (w *commentBodyWalker) validateMarks(node map[string]any, path string) {
	marks, _ := node["marks"].([]any)
	for i, raw := range marks {
		markPath := fmt.Sprintf("%s.marks[%d]", path, i)
		mark, _ := raw.(map[string]any)
		if mark == nil {
			w.multiErr.Add(markPath, errortypes.ErrInvalid, "Mark must be an object")
			continue
		}
		markType, _ := mark["type"].(string)
		if !commentInlineMarks[markType] {
			w.multiErr.Add(
				markPath,
				errortypes.ErrInvalid,
				"Mark type must be bold, italic, underline, strike, or link",
			)
			continue
		}
		if markType == commentMarkLink {
			attrs, _ := mark["attrs"].(map[string]any)
			href, _ := attrs["href"].(string)
			if !isSafeCommentLink(href) {
				w.multiErr.Add(
					markPath,
					errortypes.ErrInvalid,
					"Link must be an http or https URL",
				)
			}
		}
	}
}

func isSafeCommentLink(href string) bool {
	if href == "" || len(href) > CommentBodyMaxLink {
		return false
	}
	lower := strings.ToLower(href)
	return strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://")
}

func RenderCommentBodyText(body map[string]any) string {
	if body == nil || nodeType(body) != commentNodeDoc {
		return ""
	}

	lines := make([]string, 0, 8)
	for _, block := range nodeContent(body) {
		lines = append(lines, blockLines(block, "")...)
	}

	return strings.Join(lines, "\n")
}

func blockLines(node map[string]any, indent string) []string {
	switch nodeType(node) {
	case commentNodeParagraph:
		return []string{indent + inlineText(nodeContent(node))}
	case commentNodeBulletList:
		return listLines(node, indent, func(int) string { return "- " })
	case commentNodeOrderedList:
		start := 1
		if raw, ok := nodeAttrs(node)["start"].(float64); ok && raw >= 1 {
			start = int(raw)
		}
		return listLines(node, indent, func(i int) string {
			return strconv.Itoa(start+i) + ". "
		})
	default:
		return nil
	}
}

func listLines(node map[string]any, indent string, marker func(int) string) []string {
	lines := make([]string, 0, 4)
	itemIdx := 0
	for _, item := range nodeContent(node) {
		if nodeType(item) != commentNodeListItem {
			continue
		}
		prefix := indent + marker(itemIdx)
		continuation := indent + "  "
		first := true
		for _, child := range nodeContent(item) {
			switch nodeType(child) {
			case commentNodeParagraph:
				text := inlineText(nodeContent(child))
				if first {
					lines = append(lines, prefix+text)
					first = false
				} else {
					lines = append(lines, continuation+text)
				}
			case commentNodeBulletList, commentNodeOrderedList:
				lines = append(lines, blockLines(child, continuation)...)
				first = false
			}
		}
		if first {
			lines = append(lines, prefix)
		}
		itemIdx++
	}
	return lines
}

func inlineText(inlines []map[string]any) string {
	var sb strings.Builder
	for _, inline := range inlines {
		switch nodeType(inline) {
		case commentNodeText:
			text, _ := inline["text"].(string)
			sb.WriteString(text)
		case commentNodeMention:
			label, _ := nodeAttrs(inline)["label"].(string)
			sb.WriteByte('@')
			sb.WriteString(label)
		case commentNodeEntityRef:
			label, _ := nodeAttrs(inline)["label"].(string)
			sb.WriteByte('#')
			sb.WriteString(label)
		case commentNodeHardBreak:
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}

func ExtractMentionUserIDs(body map[string]any) []pulid.ID {
	ids := make([]pulid.ID, 0, 4)
	seen := make(map[pulid.ID]bool, 4)
	walkCommentInlines(body, func(node map[string]any) {
		if nodeType(node) != commentNodeMention {
			return
		}
		raw, _ := nodeAttrs(node)["id"].(string)
		id, err := pulid.Parse(raw)
		if err != nil || seen[id] {
			return
		}
		seen[id] = true
		ids = append(ids, id)
	})
	return ids
}

func ExtractEntityRefs(body map[string]any) []CommentEntityRef {
	refs := make([]CommentEntityRef, 0, 4)
	seen := make(map[string]bool, 4)
	walkCommentInlines(body, func(node map[string]any) {
		if nodeType(node) != commentNodeEntityRef {
			return
		}
		attrs := nodeAttrs(node)
		entityType, _ := attrs["entityType"].(string)
		if !commentEntityRefTypes[entityType] {
			return
		}
		raw, _ := attrs["entityId"].(string)
		id, err := pulid.Parse(raw)
		if err != nil {
			return
		}
		key := entityType + ":" + id.String()
		if seen[key] {
			return
		}
		seen[key] = true
		label, _ := attrs["label"].(string)
		refs = append(refs, CommentEntityRef{
			EntityType: CommentEntityRefType(entityType),
			EntityID:   id,
			Label:      label,
		})
	})
	return refs
}

func walkCommentInlines(node map[string]any, fn func(map[string]any)) {
	if node == nil {
		return
	}
	switch nodeType(node) {
	case commentNodeText, commentNodeMention, commentNodeEntityRef, commentNodeHardBreak:
		fn(node)
		return
	}
	for _, child := range nodeContent(node) {
		walkCommentInlines(child, fn)
	}
}

func nodeType(node map[string]any) string {
	t, _ := node["type"].(string)
	return t
}

func nodeContent(node map[string]any) []map[string]any {
	raw, _ := node["content"].([]any)
	if len(raw) == 0 {
		return nil
	}
	content := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		child, _ := item.(map[string]any)
		content = append(content, child)
	}
	return content
}

func nodeAttrs(node map[string]any) map[string]any {
	attrs, _ := node["attrs"].(map[string]any)
	return attrs
}
