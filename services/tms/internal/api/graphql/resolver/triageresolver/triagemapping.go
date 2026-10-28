package triageresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/loaders"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/loaderfields"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

func ruleFromInput(input *gqlmodel.TriageRuleInput) (*triage.Rule, error) {
	rule := &triage.Rule{
		Classification: input.Classification,
		SenderScope:    input.SenderScope,
		Sender:         stringutils.FromPtr(input.Sender),
		Action:         input.Action,
		Enabled:        input.Enabled == nil || *input.Enabled,
	}

	var err error
	if rule.MailboxID, err = gqlid.OptionalScopedID("mailboxId", input.MailboxID); err != nil {
		return nil, err
	}
	if rule.AssigneeID, err = gqlid.OptionalScopedID("assigneeId", input.AssigneeID); err != nil {
		return nil, err
	}

	return rule, nil
}

func userByID(ctx context.Context, id pulid.ID) (*tenant.User, error) {
	if id.IsNil() {
		return nil, nil //nolint:nilnil // no one is named, which is not a failure
	}
	if err := loaderfields.RequireLoaders(ctx); err != nil {
		return nil, err
	}

	return loaderfields.OptionalMatch(loaders.UserByID(ctx, id.String()))
}

func timestampPtr(value int64) *int {
	if value <= 0 {
		return nil
	}
	converted := int(value)

	return &converted
}
