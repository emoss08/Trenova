package checklisttest

import (
	"context"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type NoRefresh struct{}

func (NoRefresh) RefreshEvidence(context.Context, pagination.TenantInfo, pulid.ID) error {
	return nil
}
