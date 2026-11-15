package formulatemplateservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/platformcatalogports"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

var _ permissionports.PricingTemplatePlan = (*Service)(nil)

func (s *Service) ratingOnPlan(ctx context.Context) (bool, error) {
	return s.features.Enabled(ctx, platformcatalog.FeatureRating)
}

func (s *Service) OffPlan(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	templateIDs ...pulid.ID,
) (map[pulid.ID]struct{}, error) {
	chosen := chosenTemplates(templateIDs)
	if len(chosen) == 0 {
		return map[pulid.ID]struct{}{}, nil
	}

	on, err := s.ratingOnPlan(ctx)
	if err != nil || on {
		return map[pulid.ID]struct{}{}, err
	}

	own, err := s.repo.NonStandardIDs(ctx, formulatemplateports.GetFormulaTemplatesByIDsRequest{
		TenantInfo:  tenantInfo,
		TemplateIDs: chosen,
	})
	if err != nil {
		return nil, err
	}

	off := make(map[pulid.ID]struct{}, len(own))
	for _, id := range own {
		off[id] = struct{}{}
	}

	return off, nil
}

func (s *Service) RefuseOffPlan(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	templateIDs ...pulid.ID,
) error {
	off, err := s.OffPlan(ctx, tenantInfo, templateIDs...)
	if err != nil || len(off) == 0 {
		return err
	}

	return platformcatalogports.RequireFeature(ctx, s.features, platformcatalog.FeatureRating)
}

func errCustomTemplateUnreadable() error {
	return errortypes.NewAuthorizationError(
		"You do not have permission to read this formula template's inputs",
	)
}

func chosenTemplates(ids []pulid.ID) []pulid.ID {
	chosen := make([]pulid.ID, 0, len(ids))
	seen := make(map[pulid.ID]struct{}, len(ids))
	for _, id := range ids {
		if id.IsNil() {
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		chosen = append(chosen, id)
	}

	return chosen
}

type PricingInputsRequest struct {
	TenantInfo    pagination.TenantInfo
	TemplateID    pulid.ID
	MayReadCustom bool
}

type PricingInputs struct {
	Inputs []tarifftypes.InputDecl `json:"inputs"`
}

func (s *Service) PricingInputs(
	ctx context.Context,
	req *PricingInputsRequest,
) (*PricingInputs, error) {
	template, err := s.repo.GetByID(ctx, formulatemplateports.GetFormulaTemplateByIDRequest{
		TemplateID: req.TemplateID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	if !template.IsStandard() {
		if err = platformcatalogports.RequireFeature(
			ctx,
			s.features,
			platformcatalog.FeatureRating,
		); err != nil {
			return nil, err
		}
		if !req.MayReadCustom {
			return nil, errCustomTemplateUnreadable()
		}
	}

	billing, err := s.resolver.BillingVersion(ctx, template, req.TenantInfo, timeutils.NowUnix())
	if err != nil {
		return nil, err
	}

	return &PricingInputs{Inputs: billingInputs(template, billing.InForce, billing.Scheduled)}, nil
}

func billingInputs(
	template *formulatemplate.FormulaTemplate,
	versions ...*formulatemplate.FormulaTemplateVersion,
) []tarifftypes.InputDecl {
	for _, version := range versions {
		if version != nil {
			return nonNilInputs(version.Inputs)
		}
	}

	return nonNilInputs(template.Inputs)
}

func nonNilInputs(inputs []tarifftypes.InputDecl) []tarifftypes.InputDecl {
	if inputs == nil {
		return []tarifftypes.InputDecl{}
	}

	return inputs
}
