package invoice

import (
	"cmp"
	"slices"
)

func LatestSendAttempts(attempts []*EmailAttempt) []*EmailAttempt {
	ordered := make([]*EmailAttempt, 0, len(attempts))
	for _, attempt := range attempts {
		if attempt != nil {
			ordered = append(ordered, attempt)
		}
	}
	slices.SortStableFunc(ordered, func(a, b *EmailAttempt) int {
		return cmp.Or(
			cmp.Compare(a.CreatedAt, b.CreatedAt),
			cmp.Compare(a.AttemptNumber, b.AttemptNumber),
			cmp.Compare(a.ID.String(), b.ID.String()),
		)
	})
	start := 0
	for idx, attempt := range ordered {
		if attempt.AttemptNumber <= 1 {
			start = idx
		}
	}
	return ordered[start:]
}
