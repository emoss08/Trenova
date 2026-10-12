package inboundreplyresolver

import (
	"context"
	"errors"
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func permissionDeps(granted ...string) (*Deps, *resolvertest.GrantingPermissionEngine) {
	engine := &resolvertest.GrantingPermissionEngine{Granted: map[string]bool{}}
	for _, grant := range granted {
		engine.Granted[grant] = true
	}

	return &Deps{Core: &base.Core{PermissionEngine: engine}}, engine
}

func signedIn(t *testing.T) context.Context {
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

func TestReplyAttachmentResolvers_CheckTheInboxPermission(t *testing.T) {
	t.Parallel()

	messageID := pulid.MustNew("imsg_").String()
	documentID := pulid.MustNew("doc_").String()

	calls := []struct {
		name      string
		operation permission.Operation
		run       func(ctx context.Context, deps *Deps) error
	}{
		{"inboundReplyAttachables", permission.OpRead, func(ctx context.Context, deps *Deps) error {
			_, err := (&QueryResolver{deps}).InboundReplyAttachables(ctx, messageID)
			return err
		}},
		{"inboundReplyAttachmentLimit", permission.OpRead, func(ctx context.Context, deps *Deps) error {
			_, err := (&QueryResolver{deps}).InboundReplyAttachmentLimit(ctx)
			return err
		}},
		{"saveInboundReplyDraft", permission.OpUpdate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).SaveInboundReplyDraft(
				ctx,
				messageID,
				"Re: detention",
				"Attached.",
				[]string{documentID},
			)
			return err
		}},
		{"sendInboundReply", permission.OpUpdate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).SendInboundReply(ctx, gqlmodel.SendInboundReplyInput{
				MessageID:   messageID,
				Subject:     "Re: detention",
				Body:        "Attached.",
				DocumentIds: []string{documentID},
			})
			return err
		}},
	}

	for _, call := range calls {
		t.Run(call.name, func(t *testing.T) {
			t.Parallel()

			deps, engine := permissionDeps()
			err := call.run(signedIn(t), deps)

			var authzErr *errortypes.AuthorizationError
			require.True(t, errors.As(err, &authzErr), "%s must be refused: %v", call.name, err)
			require.Len(t, engine.Requests, 1)
			assert.Equal(t, permission.ResourceInboundMessage.String(), engine.Requests[0].Resource)
			assert.Equal(t, call.operation, engine.Requests[0].Operation)
		})
	}
}

func TestSendInboundReplyWithFilesAlsoNeedsCustomerCommunication(t *testing.T) {
	t.Parallel()

	deps, engine := permissionDeps(permission.ResourceInboundMessage.String() + "|" + string(permission.OpUpdate))

	_, err := (&MutationResolver{deps}).SendInboundReply(signedIn(t), gqlmodel.SendInboundReplyInput{
		MessageID:   pulid.MustNew("imsg_").String(),
		Subject:     "Re: detention",
		Body:        "Attached.",
		DocumentIds: []string{pulid.MustNew("doc_").String()},
	})

	var authzErr *errortypes.AuthorizationError
	require.True(t, errors.As(err, &authzErr), "%v", err)
	require.Len(t, engine.Requests, 2)
	assert.Equal(t, permission.ResourceCustomerCommunication.String(), engine.Requests[1].Resource)
}

func TestInboundReplyAttachmentLimitIsTheEmailBudget(t *testing.T) {
	t.Parallel()

	deps, _ := permissionDeps(permission.ResourceInboundMessage.String() + "|" + string(permission.OpRead))

	limit, err := (&QueryResolver{deps}).InboundReplyAttachmentLimit(signedIn(t))

	require.NoError(t, err)
	assert.Equal(t, int(emailports.MaxAttachmentBytes), limit)
}

func TestSaveInboundReplyDraftRefusesAMalformedDocumentID(t *testing.T) {
	t.Parallel()

	deps, _ := permissionDeps(permission.ResourceInboundMessage.String() + "|" + string(permission.OpUpdate))

	_, err := (&MutationResolver{deps}).SaveInboundReplyDraft(
		signedIn(t),
		pulid.MustNew("imsg_").String(),
		"Re: detention",
		"Attached.",
		[]string{"not-an-id"},
	)

	require.Error(t, err)
}
