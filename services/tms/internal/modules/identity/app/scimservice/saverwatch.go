package scimservice

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/scimports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// ExpiredSaverWindow is how far back the sweep looks for role assignments
// that ran out, twice the sweep's own interval so a late run misses none.
const ExpiredSaverWindow = 30 * time.Minute

type SaverWatchParams struct {
	fx.In

	Logger    *zap.Logger
	Repo      scimports.Repository
	Requester scimports.ReconcileRequester
}

// SaverWatch re-checks a directory's grants as soon as the authority behind
// them changes. A grant stands only while the person who saved its mapping
// still holds the role, so a change to that person's permissions, or to the
// organization's roles, marks the grants their mappings made for reconcile
// instead of leaving them until the next sweep.
type SaverWatch struct {
	l         *zap.Logger
	repo      scimports.Repository
	requester scimports.ReconcileRequester
}

var (
	_ permissionports.PermissionChangeListener = (*SaverWatch)(nil)
	_ tenantports.UserAccessRevoker            = (*SaverWatch)(nil)
)

func NewSaverWatch(p SaverWatchParams) *SaverWatch {
	return &SaverWatch{
		l:         p.Logger.Named("service.scim-saver-watch"),
		repo:      p.Repo,
		requester: p.Requester,
	}
}

func (w *SaverWatch) PermissionsChanged(
	ctx context.Context,
	tenant pagination.TenantInfo,
	userID pulid.ID,
) {
	filter := scimports.MappingFilter{OrganizationID: tenant.OrgID, SavedByID: userID}
	if err := w.mark(ctx, filter); err != nil {
		w.l.Error("could not mark SCIM grants for reconcile after a permission change",
			zap.String("orgID", tenant.OrgID.String()),
			zap.String("userID", userID.String()),
			zap.Error(err))
	}
}

func (w *SaverWatch) RevokeUserAccess(
	ctx context.Context,
	req tenantports.RevokeUserAccessRequest,
) error {
	return w.mark(ctx, scimports.MappingFilter{
		OrganizationID: req.OrganizationID,
		SavedByID:      req.UserID,
	})
}

func (w *SaverWatch) mark(ctx context.Context, filter scimports.MappingFilter) error {
	directories, err := w.repo.MarkMappingsPending(ctx, filter)
	if err != nil || len(directories) == 0 {
		return err
	}

	kernelports.AfterCommit(ctx, func(committed context.Context) {
		for _, directory := range directories {
			w.requester.RequestReconcile(committed, pagination.TenantInfo{
				OrgID: directory.OrganizationID,
				BuID:  directory.BusinessUnitID,
			}, directory.DirectoryID)
		}
	})

	return nil
}
