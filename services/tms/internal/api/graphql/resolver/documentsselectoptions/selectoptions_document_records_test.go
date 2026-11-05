package documentsselectoptions

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/modules/documents/app/documentservice"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type allowAll struct {
	permissionports.PermissionEngine
}

func (allowAll) Check(
	context.Context,
	*permissionports.PermissionCheckRequest,
) (*permissionports.PermissionCheckResult, error) {
	return &permissionports.PermissionCheckResult{Allowed: true}, nil
}

type viewerRecorder struct {
	documentports.DocumentRepository

	viewer pulid.ID
}

func (v *viewerRecorder) SelectOptions(
	_ context.Context,
	req *documentports.DocumentSelectOptionsRequest,
) (*pagination.ListResult[*document.Document], error) {
	v.viewer = req.Viewer

	return &pagination.ListResult[*document.Document]{}, nil
}

func TestDocumentOptions_AskForWhatTheReaderMaySee(t *testing.T) {
	t.Parallel()

	reader := pulid.MustNew("usr_")
	repo := &viewerRecorder{}
	sources := &Sources{
		Core: &resolvercore.Core{PermissionEngine: allowAll{}},
		DocumentService: documentservice.New(documentservice.Params{
			Logger: zap.NewNop(),
			Repo:   repo,
		}),
	}
	ctx := gqlctx.WithAuthContext(t.Context(), &authctx.AuthContext{
		PrincipalType:  authctx.PrincipalTypeUser,
		PrincipalID:    reader,
		UserID:         reader,
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	})

	_, err := sources.Document(ctx, selectRequest(10, 0))

	require.NoError(t, err)
	assert.Equal(t, reader, repo.viewer)
}
