package briefing_test

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/briefing"
	"github.com/stretchr/testify/assert"
)

func page() []briefing.Section {
	return []briefing.Section{
		{
			Key:     briefing.SectionToday,
			Title:   "Today",
			Summary: "142 loads are moving.",
			Path:    "/dispatch/console",
		},
		{
			Key:     briefing.SectionExceptions,
			Title:   "Needs attention",
			Summary: "6 loads may miss their window.",
			Items: []briefing.Item{
				{Label: "At risk", Value: "6", Path: "/desk/needs?lens=watchtower"},
			},
		},
		{
			Key:     briefing.SectionDecisions,
			Title:   "Decisions waiting",
			Summary: "3 decisions wait.",
			Path:    "/desk/needs?lens=decisions",
		},
		{
			Key:     briefing.SectionCash,
			Title:   "Cash received yesterday",
			Summary: "Nothing arrived.",
		},
	}
}

const headline = "142 loads are moving. 6 may miss today's appointments, and 3 decisions wait on you."

func TestLinkPhrases_OpensEachPhraseOnItsSectionInHeadlineOrder(t *testing.T) {
	links := briefing.LinkPhrases(headline, page(), []briefing.Phrase{
		{Text: "3 decisions wait on you", Section: briefing.SectionDecisions},
		{Text: "6 may miss today's appointments", Section: briefing.SectionExceptions},
	})

	assert.Equal(t, []briefing.HeadlineLink{
		{
			Text:    "6 may miss today's appointments",
			Section: briefing.SectionExceptions,
			Path:    "/desk/needs?lens=watchtower",
		},
		{
			Text:    "3 decisions wait on you",
			Section: briefing.SectionDecisions,
			Path:    "/desk/needs?lens=decisions",
		},
	}, links)
}

func TestLinkPhrases_DropsWhatCannotBeLinked(t *testing.T) {
	tests := []struct {
		name   string
		phrase briefing.Phrase
	}{
		{
			name:   "a phrase the headline does not hold",
			phrase: briefing.Phrase{Text: "7 loads", Section: briefing.SectionExceptions},
		},
		{
			name:   "a section the page does not have",
			phrase: briefing.Phrase{Text: "142 loads are moving", Section: briefing.SectionBilling},
		},
		{
			name:   "a section with nowhere to open",
			phrase: briefing.Phrase{Text: "142 loads are moving", Section: briefing.SectionCash},
		},
		{
			name:   "a key nothing defines",
			phrase: briefing.Phrase{Text: "142 loads are moving", Section: "Weather"},
		},
		{
			name:   "an empty phrase",
			phrase: briefing.Phrase{Text: "  ", Section: briefing.SectionToday},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			links := briefing.LinkPhrases(headline, page(), []briefing.Phrase{tt.phrase})

			assert.Empty(t, links)
			assert.NotNil(t, links)
		})
	}
}

func TestLinkPhrases_KeepsTheFirstOfTwoOverlappingPhrases(t *testing.T) {
	links := briefing.LinkPhrases(headline, page(), []briefing.Phrase{
		{Text: "142 loads are moving", Section: briefing.SectionToday},
		{Text: "loads are moving. 6", Section: briefing.SectionExceptions},
	})

	assert.Len(t, links, 1)
	assert.Equal(t, "142 loads are moving", links[0].Text)
}

func TestLinkPhrases_KeepsAtMostThree(t *testing.T) {
	long := "A b, c d, e f, g h."
	sections := []briefing.Section{{Key: briefing.SectionToday, Path: "/dispatch/console"}}

	links := briefing.LinkPhrases(long, sections, []briefing.Phrase{
		{Text: "A b", Section: briefing.SectionToday},
		{Text: "c d", Section: briefing.SectionToday},
		{Text: "e f", Section: briefing.SectionToday},
		{Text: "g h", Section: briefing.SectionToday},
	})

	assert.Len(t, links, briefing.MaxHeadlineLinks)
	assert.Equal(t, "e f", links[2].Text)
}

func TestComputedHeadline_LinksTheFirstSummaryToItsSection(t *testing.T) {
	text, links := briefing.ComputedHeadline(page())

	assert.Equal(t, "142 loads are moving.", text)
	assert.Equal(t, []briefing.HeadlineLink{{
		Text:    "142 loads are moving.",
		Section: briefing.SectionToday,
		Path:    "/dispatch/console",
	}}, links)
}

func TestComputedHeadline_SaysTheMorningIsQuietWithNothingComputed(t *testing.T) {
	text, links := briefing.ComputedHeadline(nil)

	assert.Equal(t, "Nothing needs you this morning.", text)
	assert.Empty(t, links)
	assert.NotNil(t, links)
}

func TestNormalize_DropsALinkTheShortenedHeadlineNoLongerHolds(t *testing.T) {
	tail := "the phrase at the end"
	entity := &briefing.Briefing{
		Headline: strings.Repeat("a", briefing.MaxHeadlineLength) + " " + tail,
		HeadlineLinks: []briefing.HeadlineLink{
			{Text: tail, Section: briefing.SectionToday, Path: "/dispatch/console"},
		},
	}

	entity.Normalize()

	assert.Empty(t, entity.HeadlineLinks)
}

func TestNormalize_DropsALinkThatLeavesTheApplication(t *testing.T) {
	entity := &briefing.Briefing{
		Headline: headline,
		HeadlineLinks: []briefing.HeadlineLink{
			{Text: "142 loads", Section: briefing.SectionToday, Path: "https://example.com"},
			{Text: "3 decisions", Section: briefing.SectionDecisions, Path: "//example.com"},
		},
	}

	entity.Normalize()

	assert.Empty(t, entity.HeadlineLinks)
}
