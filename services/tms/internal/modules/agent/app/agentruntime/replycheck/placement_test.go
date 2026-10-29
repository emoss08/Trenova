package replycheck

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestStrayArtifactLinks(t *testing.T) {
	t.Parallel()

	ref := ArtifactRef(&agentruntimeports.ShownArtifact{
		ID:    pulid.MustNew("art_"),
		Title: "Formula template Per CWT (Hundredweight)",
	})
	id := ArtifactRefIDs(ref)[0]

	stray := map[string]string{
		"after a sentence that has ended": "Its description says it multiplies a base rate " +
			"per CWT by the shipment weight divided by 100, rounded up. " + ref + "\n\n" +
			"You can view the template in Formula templates.",
		"on a line of its own":          "It is active.\n\n" + ref,
		"bold, on a line of its own":    "It is active.\n\n**" + ref + "**",
		"as a list item":                "It is active.\n\n- " + ref,
		"in a table":                    "| Template |\n| --- |\n| " + ref + " |",
		"in a heading":                  "## " + ref + "\n\nIt is active.",
		"after an arrow":                "It is active. → " + ref + ".",
		"as a sentence of its own":      ref + ". It is active.",
		"after a table":                 "| A |\n| - |\n| 1 |\n\n" + ref,
		"set off by a question mark":    "Want the rest? " + ref,
		"at the end of a list item":     "- It is active. " + ref,
		"after an exclamation mark":     "It is active! " + ref + ".",
		"between two sentences":         "It is active. " + ref + ". It rates by weight.",
		"inside a fenced code block":    "```\n" + ref + " is here\n```",
		"after a blockquote's sentence": "> It is active. " + ref,
	}
	for name, reply := range stray {
		assert.Equal(t, []string{id}, StrayArtifactLinks(reply), name)
	}

	inline := map[string]string{
		"leading a sentence":         ref + " has its full details.",
		"inside a sentence":          "See the " + ref + " for its full details.",
		"ending a sentence":          "Its full details are in " + ref + ".",
		"after a colon":              "Here is the template: " + ref + ".",
		"in a list item's sentence":  "- See " + ref + " for the rest.",
		"in a quoted sentence":       "> See " + ref + " for the rest.",
		"between two sentences":      "It is active. Read " + ref + " next. It rates by weight.",
		"with its words left out":    "Its details are in [](artifact:" + id + ").",
		"on a line after a sentence": "It is active.\nIts details are in " + ref + ".",
		"with a figure before it":    "It charges $2.50 per CWT, as " + ref + " shows.",
	}
	for name, reply := range inline {
		assert.Empty(t, StrayArtifactLinks(reply), name)
	}
}

func TestNameArtifactLinks(t *testing.T) {
	t.Parallel()

	titles := map[string]string{"art_1": "Shipment S26100090038417", "art_2": "Shipments"}
	titleOf := func(id, _ string) (string, bool) {
		title, ok := titles[id]
		return title, ok
	}

	assert.Equal(t,
		"Its [Shipment S26100090038417](artifact:art_1) has the details; "+
			"[Shipments](artifact:art_2) lists the rest, and the brief has more.",
		NameArtifactLinks("Its [shipment card](artifact:art_1) has the details; "+
			"[Shipments table](artifact:art_2) lists the rest, and the "+
			"[brief](artifact:art_9) has more.", titleOf))
	assert.Equal(t, "No links here.", NameArtifactLinks("No links here.", titleOf))
	assert.Equal(t, "[Shipments](artifact:art_2)", NameArtifactLinks("[](artifact:art_2)", titleOf))
}

func TestLinkRefs_JoinsRefsToTheFirstSentence(t *testing.T) {
	t.Parallel()

	a := "[Shipments](artifact:art_1)"
	b := "[Invoices](artifact:art_2)"
	cases := []struct {
		name string
		text string
		refs []string
		want string
	}{
		{
			name: "before the closing punctuation of the first sentence line",
			text: "Fifteen loads are late.\n\nWant me to call the carriers?",
			refs: []string{a},
			want: "Fifteen loads are late (" + a + ").\n\nWant me to call the carriers?",
		},
		{
			name: "after a colon that introduces it",
			text: "Aquí están las cargas:\n\n- PRO-1\n- PRO-2",
			refs: []string{a},
			want: "Aquí están las cargas: " + a + "\n\n- PRO-1\n- PRO-2",
		},
		{
			name: "after the markup round a colon",
			text: "**Late loads:**",
			refs: []string{a},
			want: "**Late loads:** " + a,
		},
		{
			name: "inside the markup round a sentence",
			text: "**Fifteen loads are late.**",
			refs: []string{a},
			want: "**Fifteen loads are late (" + a + ").**",
		},
		{
			name: "before a full-width stop",
			text: "有十五批货物延误。",
			refs: []string{a},
			want: "有十五批货物延误 (" + a + ")。",
		},
		{
			name: "several, in one parenthesis",
			text: "Fifteen loads are late and three invoices are open!",
			refs: []string{a, b},
			want: "Fifteen loads are late and three invoices are open (" + a + ", " + b + ")!",
		},
		{
			name: "past a label, a table, a heading and a fence",
			text: "## Late\n\n**Summary**\n\n| A |\n| - |\n| 1 |\n\n```\nnot this.\n```\n" +
				"Fifteen loads are late.",
			refs: []string{a},
			want: "## Late\n\n**Summary**\n\n| A |\n| - |\n| 1 |\n\n```\nnot this.\n```\n" +
				"Fifteen loads are late (" + a + ").",
		},
		{
			name: "to a line with no closing punctuation when none has one",
			text: "**Summary**",
			refs: []string{a},
			want: "**Summary** (" + a + ")",
		},
		{
			name: "on a line of its own when the reply has no prose",
			text: "| A |\n| - |\n| 1 |",
			refs: []string{a},
			want: "| A |\n| - |\n| 1 |\n\n" + a,
		},
		{name: "as the reply when it is empty", text: "", refs: []string{a}, want: a},
		{name: "unchanged with nothing to name", text: "It is active.", want: "It is active."},
	}
	for _, tc := range cases {
		got, _ := linkRefs(strings.Split(tc.text, "\n"), nil, tc.refs)
		assert.Equal(t, tc.want, got, tc.name)
		assert.NotContains(t, got, "The full", tc.name)
	}
}
