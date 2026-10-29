package inboundmessageresolver

import (
	"context"
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func grant(resource permission.Resource, op permission.Operation) string {
	return resource.String() + "|" + string(op)
}

func triageDeps(grants ...string) *Deps {
	engine := &resolvertest.GrantingPermissionEngine{Granted: make(map[string]bool, len(grants))}
	for _, g := range grants {
		engine.Granted[g] = true
	}

	return &Deps{Core: &resolvercore.Core{PermissionEngine: engine}}
}

func triageCtx(t *testing.T) context.Context {
	t.Helper()

	ctx := graphql.WithOperationContext(t.Context(), &graphql.OperationContext{})
	ctx = graphql.WithFieldContext(ctx, &graphql.FieldContext{})
	userID := pulid.MustNew("usr_")

	return gqlctx.WithAuthContext(ctx, &authctx.AuthContext{
		PrincipalType:  authctx.PrincipalTypeUser,
		PrincipalID:    userID,
		UserID:         userID,
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	})
}

type triageCall struct {
	name string
	run  func(ctx context.Context, deps *Deps) error
}

func triageCalls() []triageCall {
	id := pulid.MustNew("imsg_").String()
	selection := gqlmodel.TriageSelectionInput{}

	return []triageCall{
		{"reviewInboundMessages", func(ctx context.Context, d *Deps) error {
			_, err := (&MutationResolver{d}).ReviewInboundMessages(ctx, selection,
				gqlmodel.BulkReviewInboundMessagesInput{Status: inboundmessage.StatusActioned})
			return err
		}},
		{"reopenInboundMessages", func(ctx context.Context, d *Deps) error {
			_, err := (&MutationResolver{d}).ReopenInboundMessages(ctx, selection)
			return err
		}},
		{"assignInboundMessages", func(ctx context.Context, d *Deps) error {
			_, err := (&MutationResolver{d}).AssignInboundMessages(ctx, selection, nil)
			return err
		}},
		{"snoozeInboundMessages", func(ctx context.Context, d *Deps) error {
			_, err := (&MutationResolver{d}).SnoozeInboundMessages(ctx, selection,
				gqlmodel.SnoozeInboundMessagesInput{})
			return err
		}},
		{"wakeInboundMessage", func(ctx context.Context, d *Deps) error {
			_, err := (&MutationResolver{d}).WakeInboundMessage(ctx, id)
			return err
		}},
		{"waitForInboundReply", func(ctx context.Context, d *Deps) error {
			_, err := (&MutationResolver{d}).WaitForInboundReply(ctx, id, 0)
			return err
		}},
		{"stopWaitingForInboundReply", func(ctx context.Context, d *Deps) error {
			_, err := (&MutationResolver{d}).StopWaitingForInboundReply(ctx, id)
			return err
		}},
		{"unlinkInboundMessage", func(ctx context.Context, d *Deps) error {
			_, err := (&MutationResolver{d}).UnlinkInboundMessage(ctx, id,
				gqlmodel.InboundLinkedRecordShipment)
			return err
		}},
		{"readUnreadInboundMessages", func(ctx context.Context, d *Deps) error {
			_, err := (&MutationResolver{d}).ReadUnreadInboundMessages(ctx)
			return err
		}},
		{"handOffTriage", func(ctx context.Context, d *Deps) error {
			_, err := (&MutationResolver{d}).HandOffTriage(ctx, pulid.MustNew("usr_").String())
			return err
		}},
		{"sendInboundMailboxTest", func(ctx context.Context, d *Deps) error {
			_, err := (&MutationResolver{d}).SendInboundMailboxTest(ctx,
				pulid.MustNew("imbx_").String())
			return err
		}},
		{"updateTriageResponseTargets", func(ctx context.Context, d *Deps) error {
			_, err := (&MutationResolver{d}).UpdateTriageResponseTargets(ctx, nil)
			return err
		}},
		{"updateTriageSettings", func(ctx context.Context, d *Deps) error {
			_, err := (&MutationResolver{d}).UpdateTriageSettings(
				ctx,
				gqlmodel.TriageSettingsInput{},
			)
			return err
		}},
		{"createMyTriageView", func(ctx context.Context, d *Deps) error {
			_, err := (&MutationResolver{d}).CreateMyTriageView(ctx, "Acme", "acme")
			return err
		}},
		{"inboundThread", func(ctx context.Context, d *Deps) error {
			_, err := (&QueryResolver{d}).InboundThread(ctx, "abc@acme.example")
			return err
		}},
		{"myTriageViews", func(ctx context.Context, d *Deps) error {
			_, err := (&QueryResolver{d}).MyTriageViews(ctx)
			return err
		}},
		{"triageResponseTargets", func(ctx context.Context, d *Deps) error {
			_, err := (&QueryResolver{d}).TriageResponseTargets(ctx)
			return err
		}},
		{"triageSettings", func(ctx context.Context, d *Deps) error {
			_, err := (&QueryResolver{d}).TriageSettings(ctx)
			return err
		}},
		{"triageAssignees", func(ctx context.Context, d *Deps) error {
			_, err := (&QueryResolver{d}).TriageAssignees(ctx)
			return err
		}},
	}
}

func TestTriageRootResolvers_RefuseSomeoneWithoutTriageAccess(t *testing.T) {
	t.Parallel()

	for _, call := range triageCalls() {
		t.Run(call.name, func(t *testing.T) {
			t.Parallel()

			err := call.run(triageCtx(t), triageDeps())
			require.Error(t, err)
			assert.True(t, errortypes.IsAuthorizationError(err), "%s: %v", call.name, err)
		})
	}
}

func TestTriageRootResolvers_KeepOrganizationWideChangesForManagers(t *testing.T) {
	t.Parallel()

	worker := []string{
		grant(permission.ResourceInboundMessage, permission.OpRead),
		grant(permission.ResourceInboundMessage, permission.OpUpdate),
	}
	refused := map[string]bool{
		"handOffTriage":               true,
		"sendInboundMailboxTest":      true,
		"updateTriageResponseTargets": true,
		"updateTriageSettings":        true,
	}
	for _, call := range triageCalls() {
		if !refused[call.name] {
			continue
		}
		t.Run(call.name, func(t *testing.T) {
			t.Parallel()

			err := call.run(triageCtx(t), triageDeps(worker...))
			require.Error(t, err)
			assert.True(t, errortypes.IsAuthorizationError(err), "%s: %v", call.name, err)
		})
	}
}

func TestReviewInboundMessages_TeachingARuleNeedsManage(t *testing.T) {
	t.Parallel()

	teach := inboundmessage.NovaActionIgnore
	_, err := (&MutationResolver{triageDeps(
		grant(permission.ResourceInboundMessage, permission.OpUpdate),
	)}).ReviewInboundMessages(triageCtx(t), gqlmodel.TriageSelectionInput{},
		gqlmodel.BulkReviewInboundMessagesInput{
			Status: inboundmessage.StatusIgnored,
			Teach:  &teach,
		})

	require.Error(t, err)
	assert.True(t, errortypes.IsAuthorizationError(err), "%v", err)
}
