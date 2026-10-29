package benchrun

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/deskbench"
	"github.com/emoss08/trenova/internal/deskbench/benchscenario"
)

func (r *Runner) findMentions(
	ctx context.Context,
	session *deskbench.Session,
	step *benchscenario.Step,
	utterance *benchscenario.Utterance,
) error {
	for idx, query := range step.MentionFinds() {
		values, err := r.facts(ctx, session, query)
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
