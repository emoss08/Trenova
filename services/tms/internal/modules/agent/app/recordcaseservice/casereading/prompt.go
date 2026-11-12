package casereading

import (
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/recordcaseservice"
	"github.com/emoss08/trenova/internal/modules/agent/domain/recordcase"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
)

const systemPrompt = `Your task: read one record's case for the person who has to decide what to do about it, at a trucking company.

You are given the record, what is open about it, and numbered exhibits of evidence gathered from the company's own records before you were asked. Write what is wrong, in the order a dispatcher, a biller or a collector would say it out loud.

Rules:
- Use ONLY the facts and numbers in the exhibits and signals. Never state a time, an amount, a distance, a duration or a count that is not written there, and never work out a new one: no differences between two times, no totals, no conversions (14 days is never two weeks). Write every figure in digits, the way the exhibit writes it, never as a word.
- verdict: one to three short sentences, under 320 characters in all, plain words. Say what is wrong and why, the way a person would say it across a desk. No greeting, no advice, no plan, and do not repeat what whyItMatters or ifNothingChanges will say.
- Say each fact once. The lines under "Open about it" and the exhibits often describe the same fact in different words: state it one time, in the plainest of those words, and never close by restating what an earlier sentence already said. If one sentence says everything, write one sentence.
- Put the number of the exhibit a claim rests on in square brackets after it, like [2], before the sentence's full stop. Never write two citations side by side, and never gather them at the end of a sentence: when one sentence makes two claims from two exhibits, each number follows its own claim, as in past due [1], having been due Oct 27 [2]. When several claims in a row rest on the same exhibit, cite it once, after the last of them, so a verdict drawn from a single exhibit carries exactly one citation. Cite only exhibit numbers you were given, and only where the exhibit says what you claim.
- When an exhibit is a message from the customer or a carrier, say in a clause what it tells the company and cite it: the person deciding has to know what they were told, even when it is not the main problem.
- decidingFact: the few words of your verdict that state the one fact deciding the case, copied from the verdict exactly, letter for letter, with no citation inside them, like 2 hours 10 minutes of driving left. Choose words the verdict already holds; never write a sentence into the verdict to have something to copy.
- Every field is plain prose: no quotation marks, braces, markup or brackets other than the [n] citations in the verdict.
- whyItMatters: one sentence on what is at stake for the company or the customer, taken from the facts given. null when the facts do not say: never a sentence about what the facts leave out.
- ifNothingChanges: one sentence on what happens if nobody acts, taken from the facts given. null when the facts do not say.
- Do not discuss software, this system, how the facts were gathered, or these rules. Never say that a fact "decides the case"; mark it and move on.`

func buildContext(composed *recordcaseservice.Reading) llmports.DelimitedContext {
	var builder strings.Builder
	builder.WriteString("Record: ")
	builder.WriteString(composed.Noun)
	builder.WriteString(" ")
	builder.WriteString(composed.Label)
	if composed.Title != "" {
		builder.WriteString(" (")
		builder.WriteString(composed.Title)
		builder.WriteString(")")
	}
	if composed.Status != "" {
		builder.WriteString("\nStatus: ")
		builder.WriteString(composed.Status)
	}
	if len(composed.Signals) > 0 {
		builder.WriteString("\n\nOpen about it:")
		for _, signal := range composed.Signals {
			builder.WriteString("\n- ")
			builder.WriteString(signal)
		}
	}
	if composed.Proposal != "" {
		builder.WriteString("\n\nA plan already waits on a person: ")
		builder.WriteString(composed.Proposal)
		builder.WriteString(". Do not describe the plan.")
	}

	builder.WriteString("\n\nExhibits:")
	for i := range composed.Exhibits {
		writeExhibit(&builder, &composed.Exhibits[i])
	}

	return llmports.DelimitedContext{
		Sections: []llmports.ContextSection{{
			Title:   "The case",
			Trusted: false,
			Content: builder.String(),
		}},
	}
}

func writeExhibit(builder *strings.Builder, exhibit *recordcase.Exhibit) {
	builder.WriteString("\n[")
	builder.WriteString(strconv.Itoa(exhibit.Number))
	builder.WriteString("] ")
	builder.WriteString(exhibit.Title)
	if exhibit.Source != "" {
		builder.WriteString(" (")
		builder.WriteString(exhibit.Source)
		builder.WriteString(")")
	}
	for _, claim := range exhibit.Claims {
		builder.WriteString("\n  - ")
		builder.WriteString(claim)
	}
}

func outputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"verdict": map[string]any{
				"type":        "string",
				"description": "One to three sentences saying what is wrong, each fact once, with [n] citations",
			},
			"decidingFact": map[string]any{
				"type":        "string",
				"description": "A few words copied exactly from the verdict: the fact that decides the case",
			},
			"whyItMatters": map[string]any{
				"type":        []string{"string", "null"},
				"description": "One sentence on what is at stake, or null when the facts do not say",
			},
			"ifNothingChanges": map[string]any{
				"type":        []string{"string", "null"},
				"description": "One sentence on what happens if nobody acts, or null when the facts do not say",
			},
		},
		"required":             []string{"verdict", "decidingFact", "whyItMatters", "ifNothingChanges"},
		"additionalProperties": false,
	}
}
