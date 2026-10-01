package base

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/loaders"
	"github.com/emoss08/trenova/internal/core/domain/capture"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/captureservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

// captureRecord names the record a capture points at, for a reader who could
// open it. A record the reader cannot see, or one that is gone, is absent: the
// row still carries its kind and id, and that is all it may show.
func (r *Resolver) CaptureRecord(
	ctx context.Context,
	resourceType string,
	id *pulid.ID,
) (*repositories.CaptureRecordLabel, error) {
	if resourceType == "" || id == nil || id.IsNil() ||
		!capture.IsFileableResource(resourceType) {
		return nil, nil //nolint:nilnil // no record is a valid answer
	}

	authCtx, err := r.RequireAuth(ctx)
	if err != nil {
		return nil, err
	}
	if !r.HasPermission(ctx, authCtx, permission.Resource(resourceType), permission.OpRead) {
		return nil, nil //nolint:nilnil // a record out of reach is left absent
	}

	l, err := RequestLoaders(ctx)
	if err != nil {
		return nil, err
	}

	return OptionalMatch(
		l.CaptureRecordLabel.Load(ctx, loaders.CaptureRecordKey(resourceType, *id)),
	)
}

// maxCaptureDevices bounds a device list. A person pairs a machine or two; an
// organization's whole fleet is a few hundred at most.
const maxCaptureDevices = 500

func (r *Resolver) ListCaptureDevices(
	ctx context.Context,
	authCtx *authctx.AuthContext,
	mine bool,
	status *capture.DeviceStatus,
	query *string,
) ([]*capture.CaptureDevice, error) {
	tenant := TenantInfo(authCtx)
	req := &captureservice.ListDevicesRequest{
		TenantInfo: tenant,
		Filter: &pagination.QueryOptions{
			TenantInfo: tenant,
			Pagination: pagination.Info{Limit: maxCaptureDevices},
			Query:      stringutils.FromPtr(query),
		},
		Mine: mine,
	}
	if status != nil {
		req.Status = *status
	}

	result, err := r.CaptureService.ListDevices(ctx, req)
	if err != nil {
		return nil, err
	}

	return result.Items, nil
}
