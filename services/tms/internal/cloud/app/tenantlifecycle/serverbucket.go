package tenantlifecycle

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/cloud/domain/tenantarchive"
	"github.com/emoss08/trenova/internal/cloud/ports/cloudtenantports"
)

const (
	reasonNoServerBucket = "this server holds no nightly archive storage, so it has no bucket of its own to take an archive, restore or import from; the control plane names it once the bucket is recorded"
	reasonOtherBucket    = "the target names another bucket than this server's own nightly archive storage; an archive, restore or import goes only to the server's own bucket"
)

func (s *Service) requireServerBucket(ctx context.Context, target *tenantarchive.Location) error {
	nightly, err := s.NightlyAccess(ctx)
	if errors.Is(err, cloudtenantports.ErrGrantMissing) {
		return tenantarchive.Refuse(tenantarchive.RefusalLocation, reasonNoServerBucket)
	}
	if err != nil {
		return err
	}
	if !target.SameBucket(&nightly.Location) {
		return tenantarchive.Refuse(tenantarchive.RefusalLocation, reasonOtherBucket)
	}
	return nil
}
