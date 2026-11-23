package desktranscriptresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/desktranscript"
	"github.com/emoss08/trenova/internal/modules/agent/ports/desktranscriptports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type countingService struct {
	calls int
}

func (c *countingService) ListConversations(
	context.Context,
	*desktranscriptports.ListRequest,
) (*pagination.CursorListResult[*desktranscript.Conversation], error) {
	c.calls++
	return &pagination.CursorListResult[*desktranscript.Conversation]{}, nil
}

func (c *countingService) OpenConversation(
	context.Context,
	desktranscriptports.ConversationRequest,
) (*desktranscriptports.OpenedConversation, error) {
	c.calls++
	return &desktranscriptports.OpenedConversation{}, nil
}

func (c *countingService) DownloadConversation(
	context.Context,
	desktranscriptports.ConversationRequest,
) (*kernelports.TranscriptFile, error) {
	c.calls++
	return &kernelports.TranscriptFile{}, nil
}

func (c *countingService) PreviewExport(
	context.Context,
	*desktranscriptports.PreviewRequest,
) (*desktranscriptports.ExportPreview, error) {
	c.calls++
	return &desktranscriptports.ExportPreview{}, nil
}

func (c *countingService) RequestExport(
	context.Context,
	*desktranscriptports.RequestExportRequest,
) (*desktranscript.Export, error) {
	c.calls++
	return &desktranscript.Export{}, nil
}

func (c *countingService) ListExports(
	context.Context,
	*desktranscriptports.ListExportsServiceRequest,
) (*pagination.CursorListResult[*desktranscript.Export], error) {
	c.calls++
	return &pagination.CursorListResult[*desktranscript.Export]{}, nil
}

func (c *countingService) ExportDownload(
	context.Context,
	desktranscriptports.ExportDownloadRequest,
) (*desktranscriptports.ExportDownload, error) {
	c.calls++
	return &desktranscriptports.ExportDownload{}, nil
}

func resolvers(
	granted ...permission.Operation,
) (*QueryResolver, *MutationResolver, *countingService) {
	engine := &resolvertest.GrantingPermissionEngine{Granted: make(map[string]bool)}
	for _, op := range granted {
		engine.Granted[permission.ResourceDeskTranscript.String()+"|"+string(op)] = true
	}
	service := &countingService{}
	deps := &Deps{
		Core:                  &resolvercore.Core{PermissionEngine: engine},
		DeskTranscriptService: service,
	}

	return &QueryResolver{deps}, &MutationResolver{deps}, service
}

func personCtx(t *testing.T) context.Context {
	t.Helper()
	userID := pulid.MustNew("usr_")
	return gqlctx.WithAuthContext(t.Context(), &authctx.AuthContext{
		PrincipalType:  authctx.PrincipalTypeUser,
		PrincipalID:    userID,
		UserID:         userID,
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	})
}

func takeOut(ctx context.Context, q *QueryResolver, m *MutationResolver) []error {
	id := pulid.MustNew("athr_").String()
	_, previewErr := q.DeskTranscriptExportPreview(ctx, gqlmodel.DeskConversationFilterInput{})
	_, requestErr := m.RequestDeskTranscriptExport(ctx, gqlmodel.RequestDeskTranscriptExportInput{
		ConversationIds: []string{id},
	})
	_, downloadErr := m.DeskTranscriptExportDownload(ctx, pulid.MustNew("dtex_").String())

	return []error{previewErr, requestErr, downloadErr}
}

func TestViewAloneListsAndOpensAndTakesNothingOut(t *testing.T) {
	t.Parallel()

	q, m, service := resolvers(permission.OpRead)
	ctx := personCtx(t)

	_, err := q.DeskConversations(ctx, gqlmodel.DataTableConnectionInput{}, nil)
	require.NoError(t, err)
	_, err = m.OpenDeskConversation(ctx, pulid.MustNew("athr_").String())
	require.NoError(t, err)
	_, err = q.DeskTranscriptExports(ctx, gqlmodel.DataTableConnectionInput{})
	require.NoError(t, err)
	assert.Equal(t, 3, service.calls)

	for _, err = range takeOut(ctx, q, m) {
		require.Error(t, err)
		assert.True(t, errortypes.IsAuthorizationError(err), err)
	}
	assert.Equal(t, 3, service.calls, "nothing that takes a transcript out reached the service")
}

func TestWithoutThePermissionEverythingIsRefused(t *testing.T) {
	t.Parallel()

	q, m, service := resolvers()
	ctx := personCtx(t)

	_, listErr := q.DeskConversations(ctx, gqlmodel.DataTableConnectionInput{}, nil)
	_, openErr := m.OpenDeskConversation(ctx, pulid.MustNew("athr_").String())
	_, exportsErr := q.DeskTranscriptExports(ctx, gqlmodel.DataTableConnectionInput{})
	for _, err := range append([]error{listErr, openErr, exportsErr}, takeOut(ctx, q, m)...) {
		require.Error(t, err)
		assert.True(t, errortypes.IsAuthorizationError(err), err)
	}
	assert.Zero(t, service.calls)
}

func TestTheListRefusesTheTablesOwnFilters(t *testing.T) {
	t.Parallel()

	q, _, service := resolvers(permission.OpRead)
	value := "Notes"
	_, err := q.DeskConversations(personCtx(t), gqlmodel.DataTableConnectionInput{
		FieldFilters: []*gqlmodel.FieldFilterInput{{Field: "pinnedFacts", Value: value}},
	}, nil)

	require.Error(t, err)
	assert.Zero(
		t,
		service.calls,
		"a filter on a column the page does not offer never reaches the database",
	)
}
