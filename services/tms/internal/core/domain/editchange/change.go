// Package editchange names what differs between two saves of something a
// person edits, in the words of the editor they changed it in. Version
// histories summarise saves with it and a save that lost a race uses it to
// say what the other save changed.
package editchange

import "strings"

// Change is one setting that differs. Field is the editor's own field name,
// so the editor can mark the section the setting belongs to.
type Change struct {
	Field string `json:"field"`
	Label string `json:"label"`
}

// Rule compares one setting of T. Settings an editor shows as one control,
// even when they are stored as several columns, are one rule.
type Rule[T any] struct {
	Field string
	Label string
	Same  func(a, b *T) bool
}

// Detect lists the settings that differ between two saves, in the order of
// the rules. A nil side has nothing to compare and so no changes.
func Detect[T any](rules []Rule[T], before, after *T) []Change {
	if before == nil || after == nil {
		return nil
	}

	changes := make([]Change, 0, len(rules))
	for i := range rules {
		if !rules[i].Same(before, after) {
			changes = append(changes, Change{Field: rules[i].Field, Label: rules[i].Label})
		}
	}

	return changes
}

// Summary is the one line a saved version carries in its history.
func Summary(changes []Change) string {
	if len(changes) == 0 {
		return ""
	}

	labels := make([]string, 0, len(changes))
	for _, change := range changes {
		labels = append(labels, change.Label)
	}

	return strings.Join(labels, ", ")
}
