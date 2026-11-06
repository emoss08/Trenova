package chainsnap

import (
	"regexp"
	"slices"
	"strings"
)

var aliasPattern = regexp.MustCompile(`\$[a-zA-Z][a-zA-Z0-9]*(?:[._][a-zA-Z0-9]+)*`)

func Suspects(snapshot *Snapshot, diff Diff) []string {
	index := slices.IndexFunc(snapshot.Steps, func(step StepRecord) bool { return step.ID == diff.Step })
	if index < 0 {
		return nil
	}
	drifted := snapshot.Steps[index]
	if drifted.Write {
		return []string{stepLabel(drifted)}
	}

	aliases := map[string]bool{}
	for _, value := range []any{drifted.Request, diff.Want, diff.Got} {
		for _, alias := range aliasPattern.FindAllString(short(value), -1) {
			aliases[alias] = true
		}
	}

	var suspects []string
	var nearest string
	for i := index - 1; i >= 0; i-- {
		step := snapshot.Steps[i]
		if !step.Write {
			continue
		}
		if nearest == "" {
			nearest = stepLabel(step)
		}
		text := short(step.Request) + short(step.Response)
		for alias := range aliases {
			if strings.Contains(text, alias) {
				suspects = append(suspects, stepLabel(step))
				break
			}
		}
	}
	if len(suspects) == 0 && nearest != "" {
		return []string{nearest + " (the nearest write; no shared record)"}
	}

	return suspects
}

func stepLabel(step StepRecord) string {
	return step.ID + " (" + step.Call + ")"
}
