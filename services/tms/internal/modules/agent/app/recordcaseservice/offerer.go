package recordcaseservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/recordcase"
	"github.com/emoss08/trenova/internal/modules/agent/ports/recordcaseports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

type Offerer struct {
	repo recordcaseports.RecordCaseRepository
}

func NewOfferer(repo recordcaseports.RecordCaseRepository) recordcaseports.OptionOfferer {
	return &Offerer{repo: repo}
}

func (o *Offerer) OfferOptions(
	ctx context.Context,
	req *recordcaseports.OfferOptionsRequest,
) ([]*recordcase.Option, error) {
	multiErr := errortypes.NewMultiError()
	for position, option := range req.Options {
		option.OrganizationID = req.TenantInfo.OrgID
		option.BusinessUnitID = req.TenantInfo.BuID
		option.RecordKind = req.Kind
		option.RecordID = req.ID
		option.Position = position
		option.AgentDefinitionID = req.AgentDefinitionID
		option.RunID = req.RunID
		option.Chosen = option.Recommended
		option.DraftRunID, option.DraftRequestedAt = pulid.Nil, nil
		if option.Recommended {
			option.DraftRunID = req.RunID
		}
		option.Normalize()
		option.Validate(multiErr)
	}
	recordcase.ValidateSet(req.Options, multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	return o.repo.ReplaceOptions(ctx, &recordcaseports.ReplaceOptionsRequest{
		TenantInfo: req.TenantInfo,
		RecordID:   req.ID,
		Options:    req.Options,
	})
}
