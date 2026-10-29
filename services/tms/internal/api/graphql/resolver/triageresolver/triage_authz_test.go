package triageresolver

import (
	"context"
	"errors"
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func refusingDeps() (*Deps, *resolvertest.GrantingPermissionEngine) {
	engine := &resolvertest.GrantingPermissionEngine{Granted: map[string]bool{}}

	return &Deps{Core: &resolvercore.Core{PermissionEngine: engine}}, engine
}

func authedContext(t *testing.T) context.Context {
	t.Helper()

	userID := pulid.MustNew("usr_")
	ctx := graphql.WithOperationContext(t.Context(), &graphql.OperationContext{})
	ctx = graphql.WithFieldContext(ctx, &graphql.FieldContext{})

	return gqlctx.WithAuthContext(ctx, &authctx.AuthContext{
		PrincipalType:  authctx.PrincipalTypeUser,
		PrincipalID:    userID,
		UserID:         userID,
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	})
}

func TestTriageRootResolvers_CheckTheInboxPermission(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("imsg_").String()
	ruleID := pulid.MustNew("trl_").String()
	input := gqlmodel.TriageRuleInput{
		Classification: inboundmessage.ClassificationInvoice,
		SenderScope:    triage.SenderAnyone,
		Action:         inboundmessage.NovaActionIgnore,
	}

	calls := []struct {
		name      string
		operation permission.Operation
		run       func(ctx context.Context, deps *Deps) error
	}{
		{"createTriageRule", permission.OpManage, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).CreateTriageRule(ctx, input)
			return err
		}},
		{"updateTriageRule", permission.OpManage, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).UpdateTriageRule(ctx, ruleID, 1, input)
			return err
		}},
		{"setTriageRuleEnabled", permission.OpManage, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).SetTriageRuleEnabled(ctx, ruleID, false)
			return err
		}},
		{"deleteTriageRule", permission.OpManage, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).DeleteTriageRule(ctx, ruleID)
			return err
		}},
		{
			"correctInboundClassification",
			permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).CorrectInboundClassification(
					ctx,
					id,
					inboundmessage.ClassificationTender,
				)
				return err
			},
		},
		{"flagInboundFact", permission.OpUpdate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).FlagInboundFact(ctx, id, "rate")
			return err
		}},
		{
			"resolveInboundConflict",
			permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).ResolveInboundConflict(
					ctx,
					id,
					"dup:x",
					inboundmessage.ConflictKeepBoth,
				)
				return err
			},
		},
		{"reopenInboundConflict", permission.OpUpdate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).ReopenInboundConflict(ctx, id, "dup:x")
			return err
		}},
		{"triageBriefing", permission.OpRead, func(ctx context.Context, deps *Deps) error {
			_, err := (&QueryResolver{deps}).TriageBriefing(ctx)
			return err
		}},
		{"triageRules", permission.OpRead, func(ctx context.Context, deps *Deps) error {
			_, err := (&QueryResolver{deps}).TriageRules(ctx)
			return err
		}},
		{"triageRulePreview", permission.OpRead, func(ctx context.Context, deps *Deps) error {
			_, err := (&QueryResolver{deps}).TriageRulePreview(ctx, input)
			return err
		}},
		{"triageRuleMatches", permission.OpRead, func(ctx context.Context, deps *Deps) error {
			_, err := (&QueryResolver{deps}).TriageRuleMatches(ctx, ruleID)
			return err
		}},
		{"triageSenderDomains", permission.OpRead, func(ctx context.Context, deps *Deps) error {
			_, err := (&QueryResolver{deps}).TriageSenderDomains(ctx, nil)
			return err
		}},
		{"inboundSenderProfile", permission.OpRead, func(ctx context.Context, deps *Deps) error {
			_, err := (&QueryResolver{deps}).InboundSenderProfile(ctx, id)
			return err
		}},
	}

	for _, call := range calls {
		t.Run(call.name, func(t *testing.T) {
			t.Parallel()

			deps, engine := refusingDeps()
			err := call.run(authedContext(t), deps)

			var authzErr *errortypes.AuthorizationError
			require.True(t, errors.As(err, &authzErr), "%s must be refused: %v", call.name, err)
			require.Len(t, engine.Requests, 1)
			assert.Equal(t, permission.ResourceInboundMessage.String(), engine.Requests[0].Resource)
			assert.Equal(t, call.operation, engine.Requests[0].Operation)
		})
	}
}

func TestTriageRootResolvers_NeedASignedInPerson(t *testing.T) {
	t.Parallel()

	deps, engine := refusingDeps()

	_, err := (&QueryResolver{deps}).TriageBriefing(t.Context())

	var authnErr *errortypes.AuthenticationError
	require.True(t, errors.As(err, &authnErr), "%v", err)
	assert.Empty(t, engine.Requests)
}
