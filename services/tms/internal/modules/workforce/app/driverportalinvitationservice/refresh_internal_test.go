package driverportalinvitationservice

import (
	"context"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type noRefresh struct{}

func (noRefresh) RefreshEvidence(context.Context, pagination.TenantInfo, pulid.ID) error {
	return nil
}
