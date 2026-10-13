package deskbench

import (
	"context"
	"fmt"
)

func (b *Bench) findMentions(
	ctx context.Context,
	session *Session,
	step *Step,
	utterance *Utterance,
) error {
	for idx, query := range step.mentionFinds() {
		values, err := b.FactValues(ctx, session, query)
		if err != nil {
			return fmt.Errorf("find mention %d: %w", idx+1, err)
		}
		if len(values) != 1 {
			return fmt.Errorf("find mention %d: the query returned %d rows, not one", idx+1,
				len(values))
		}
		utterance.Mentions[idx].ID = values[0]
	}

	return nil
}
