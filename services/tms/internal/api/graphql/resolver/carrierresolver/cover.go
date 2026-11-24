package carrierresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/modules/operations/ports/carriercoverports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/timeutils"
)

const defaultCoverDays = 30

func coverEndingRequest(
	now int64,
	withinDays, limit *int,
	readsShipments bool,
) (*carriercoverports.EndingRequest, error) {
	days := defaultCoverDays
	if withinDays != nil {
		days = *withinDays
	}
	if days < 1 || days > carriercoverports.MaxWithinDays {
		return nil, errortypes.NewValidationError(
			"withinDays", errortypes.ErrInvalid, "Within days must be from 1 to 90",
		)
	}
	count := carriercoverports.DefaultCarriers
	if limit != nil {
		count = *limit
	}
	if count < 1 || count > carriercoverports.MaxCarriers {
		return nil, errortypes.NewValidationError(
			"limit", errortypes.ErrInvalid, "Limit must be from 1 to 100",
		)
	}

	return &carriercoverports.EndingRequest{
		Now:   now,
		Until: now + int64(days)*timeutils.SecondsPerDay,
		Limit: count,
		Loads: readsShipments,
	}, nil
}

func coverEndingsToModel(
	covers []*carriercoverports.Cover,
	readsShipments bool,
) []*gqlmodel.CarrierCoverEnding {
	out := make([]*gqlmodel.CarrierCoverEnding, 0, len(covers))
	for _, cover := range covers {
		if cover.Policy == nil {
			continue
		}
		model := &gqlmodel.CarrierCoverEnding{
			ID:                 cover.CarrierID.String(),
			Name:               cover.Name,
			Code:               cover.Code,
			ComplianceStatus:   cover.ComplianceStatus,
			PolicyType:         cover.Policy.PolicyType,
			PolicyNumber:       cover.Policy.PolicyNumber,
			EndsAt:             int(cover.Policy.ExpirationDate),
			MissingPolicyTypes: append([]carrier.InsurancePolicyType{}, cover.Missing...),
		}
		if readsShipments && cover.LoadsReadable() {
			loads := int(cover.OpenLoads)
			model.OpenLoads = &loads
		}
		out = append(out, model)
	}

	return out
}
