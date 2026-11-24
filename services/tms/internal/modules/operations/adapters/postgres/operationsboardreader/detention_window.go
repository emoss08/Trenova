package operationsboardreader

import (
	"time"

	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

const facilityDetentionLimit = 120

type DetentionParams struct {
	fx.In

	Intelligence detentionports.DetentionIntelligenceRepository
}

type detentionAsk struct {
	tenant    pagination.TenantInfo
	period    boardcatalog.Period
	now       int64
	location  *time.Location
	locations []pulid.ID
}

type detentionWindow struct {
	first time.Time
	last  time.Time
	req   *detentionports.IntelligenceRequest
}

func (a *detentionAsk) window() (*detentionWindow, bool) {
	loc := a.location
	if loc == nil || loc == time.Local {
		loc = time.UTC
	}
	first, last, ok := a.period.Days(a.now, loc)
	if !ok {
		return nil, false
	}
	limit := facilityDetentionLimit
	if len(a.locations) > 0 {
		limit = len(a.locations)
	}

	return &detentionWindow{
		first: first,
		last:  last,
		req: &detentionports.IntelligenceRequest{
			TenantInfo:  a.tenant,
			Current:     detentionports.NewIntelligenceWindow(first, last),
			Timezone:    loc.String(),
			LocationIDs: a.locations,
			Limit:       limit,
		},
	}, true
}
