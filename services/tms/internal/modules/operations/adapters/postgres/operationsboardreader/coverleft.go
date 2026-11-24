package operationsboardreader

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/operations/ports/carriercoverports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/modules/parties/domain/partiesboard"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardreading"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardwording"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/boardreaderports"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

var carrierKind = recordkind.Kind(permission.ResourceCarrier)

type CoverParams struct {
	fx.In

	Cover carriercoverports.Reader
}

type CoverLeft struct {
	cover carriercoverports.Reader
}

func NewCoverLeft(p CoverParams) *CoverLeft {
	return &CoverLeft{cover: p.Cover}
}

func (*CoverLeft) Kind() recordkind.Kind { return carrierKind }

func (*CoverLeft) Measure() string { return partiesboard.MeasureCoverLeft }

func (r *CoverLeft) Readings(
	ctx context.Context,
	req *boardreaderports.RecordRequest,
) (map[pulid.ID]boardreading.Reading, error) {
	if len(req.IDs) == 0 {
		return map[pulid.ID]boardreading.Reading{}, nil
	}
	covers, err := r.cover.Covers(ctx, &carriercoverports.CoversRequest{
		IDs:   req.IDs,
		Loads: req.Place,
	})
	if err != nil {
		return nil, err
	}
	out := make(map[pulid.ID]boardreading.Reading, len(covers))
	for id, cover := range covers {
		if reading, ok := coverLeft(ctx, cover, req); ok {
			out[id] = reading
		}
	}

	return out, nil
}

func coverLeft(
	ctx context.Context,
	cover *carriercoverports.Cover,
	req *boardreaderports.RecordRequest,
) (boardreading.Reading, bool) {
	if cover.Policy == nil || len(cover.Missing) > 0 {
		return boardreading.Reading{}, false
	}
	ending, ended := policyEnds(ctx, cover.Policy.PolicyType)
	countdown := boardwording.Countdown{
		Ends:     cover.Policy.ExpirationDate,
		Span:     carrier.InsuranceWarningWindowSeconds,
		Date:     true,
		Location: req.Location,
		Ending:   ending,
		Ended:    ended,
	}
	reading := countdown.Read(ctx, req.Now)
	if cover.LoadsReadable() {
		reading.Fact.Place = loadsWithUs(ctx, cover.OpenLoads)
	}

	return reading, true
}

func loadsWithUs(ctx context.Context, loads int64) string {
	return i18n.T(ctx,
		"{0, plural, =0 {with none of our loads now} one {with # of our loads now} "+
			"other {with # of our loads now}}", loads)
}

func policyEnds(
	ctx context.Context,
	policyType carrier.InsurancePolicyType,
) (ending, ended string) {
	switch policyType {
	case carrier.InsurancePolicyTypeAutoLiability:
		return i18n.T(ctx, "Auto liability ends {0}"), i18n.T(ctx, "Auto liability ended {0}")
	case carrier.InsurancePolicyTypeCargoLiability:
		return i18n.T(ctx, "Cargo liability ends {0}"), i18n.T(ctx, "Cargo liability ended {0}")
	case carrier.InsurancePolicyTypeGeneralLiability:
		return i18n.T(ctx, "General liability ends {0}"),
			i18n.T(ctx, "General liability ended {0}")
	case carrier.InsurancePolicyTypeWorkersComp:
		return i18n.T(ctx, "Workers comp ends {0}"), i18n.T(ctx, "Workers comp ended {0}")
	case carrier.InsurancePolicyTypeUmbrella:
		return i18n.T(ctx, "Umbrella ends {0}"), i18n.T(ctx, "Umbrella ended {0}")
	}

	return i18n.T(ctx, "Cover ends {0}"), i18n.T(ctx, "Cover ended {0}")
}
