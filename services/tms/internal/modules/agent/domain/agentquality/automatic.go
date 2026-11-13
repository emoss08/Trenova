package agentquality

import (
	"cmp"
	"slices"

	"github.com/emoss08/trenova/shared/pulid"
)

type AutomaticCase struct {
	ID        pulid.ID
	Tool      string
	CreatedAt int64
}

func newerAutomatic(a, b AutomaticCase) int {
	return cmp.Or(
		cmp.Compare(b.CreatedAt, a.CreatedAt),
		cmp.Compare(b.ID.String(), a.ID.String()),
	)
}

func AutomaticOverflow(cases []AutomaticCase, keep int) []pulid.ID {
	if keep < 0 {
		keep = 0
	}
	if len(cases) <= keep {
		return []pulid.ID{}
	}

	type ranked struct {
		AutomaticCase
		rank int
	}

	newest := slices.Clone(cases)
	slices.SortFunc(newest, newerAutomatic)

	seen := make(map[string]int, len(newest))
	order := make([]ranked, 0, len(newest))
	for _, evalCase := range newest {
		order = append(order, ranked{AutomaticCase: evalCase, rank: seen[evalCase.Tool]})
		seen[evalCase.Tool]++
	}
	slices.SortStableFunc(order, func(a, b ranked) int {
		return cmp.Compare(a.rank, b.rank)
	})

	overflow := make([]pulid.ID, 0, len(order)-keep)
	for _, evalCase := range order[keep:] {
		overflow = append(overflow, evalCase.ID)
	}

	return overflow
}
