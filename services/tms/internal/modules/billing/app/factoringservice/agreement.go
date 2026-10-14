package factoringservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/factoringcompanyports"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const maxAgreementDocumentTypes = 20

type AgreementParams struct {
	fx.In

	DB        kernelports.DBConnection
	Repo      factoringports.AgreementRepository
	Factors   factoringcompanyports.FactoringCompanyRepository
	Requester factoringports.EvaluationRequester `optional:"true"`
	Audit     auditports.AuditService
	Realtime  agentports.RealtimeService `optional:"true"`
	Logger    *zap.Logger
}

type AgreementService struct {
	db        kernelports.DBConnection
	repo      factoringports.AgreementRepository
	factors   factoringcompanyports.FactoringCompanyRepository
	requester factoringports.EvaluationRequester
	rec       recorder
	now       func() int64
}

var _ factoringports.AgreementService = (*AgreementService)(nil)

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
func NewAgreementService(p AgreementParams) factoringports.AgreementService {
	l := p.Logger.Named("service.factoring-agreement")
	return &AgreementService{
		db:        p.DB,
		repo:      p.Repo,
		factors:   p.Factors,
		requester: p.Requester,
		rec:       recorder{audit: p.Audit, realtime: p.Realtime, l: l},
		now:       timeutils.NowUnix,
	}
}

func (s *AgreementService) List(
	ctx context.Context,
	req factoringports.ListAgreementsRequest,
) ([]*factoring.Agreement, error) {
	if req.FactoringCompanyID.IsNil() {
		return nil, errortypes.NewValidationError(
			"factoringCompanyId",
			errortypes.ErrRequired,
			"Factoring company is required",
		)
	}
	return s.repo.ListByFactoringCompany(ctx, req)
}

func (s *AgreementService) Get(
	ctx context.Context,
	req factoringports.GetAgreementRequest,
) (*factoring.Agreement, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *AgreementService) Create(
	ctx context.Context,
	req *factoringports.CreateAgreementRequest,
	actor *permissionports.RequestActor,
) (*factoring.Agreement, error) {
	if err := requireRequest(req != nil); err != nil {
		return nil, err
	}
	if err := requireActor(actor, "Creating a factoring agreement"); err != nil {
		return nil, err
	}

	entity := &factoring.Agreement{
		OrganizationID:     req.TenantInfo.OrgID,
		BusinessUnitID:     req.TenantInfo.BuID,
		FactoringCompanyID: req.Input.FactoringCompanyID,
		Status:             factoring.AgreementStatusDraft,
	}
	applyAgreementInput(entity, &req.Input)
	if err := s.validate(ctx, entity, &req.Input); err != nil {
		return nil, err
	}

	created, err := s.repo.Create(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.rec.log(actor, &auditEntry{
		resource: permission.ResourceFactoringAgreement,
		entity:   created,
		op:       permission.OpCreate,
		comment:  "Factoring agreement drafted",
	})
	s.rec.publish(ctx, permission.ResourceFactoringAgreement, created, actor, "created")
	return created, nil
}

func (s *AgreementService) Update(
	ctx context.Context,
	req *factoringports.UpdateAgreementRequest,
	actor *permissionports.RequestActor,
) (*factoring.Agreement, error) {
	if err := requireRequest(req != nil); err != nil {
		return nil, err
	}
	if err := requireActor(actor, "Changing a factoring agreement"); err != nil {
		return nil, err
	}

	var previous, updated *factoring.Agreement
	err := s.db.WithTx(ctx, kernelports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		current, err := s.repo.GetByID(txCtx, factoringports.GetAgreementRequest{
			ID:         req.ID,
			TenantInfo: req.TenantInfo,
		})
		if err != nil {
			return err
		}
		if current.Status == factoring.AgreementStatusEnded {
			return errortypes.NewBusinessError("An ended agreement cannot be changed")
		}
		if !req.Input.FactoringCompanyID.IsNil() &&
			req.Input.FactoringCompanyID != current.FactoringCompanyID {
			return errortypes.NewValidationError(
				"factoringCompanyId",
				errortypes.ErrInvalid,
				"An agreement cannot move to another factor; draft a new one instead",
			)
		}
		previous = cloneAgreement(current)

		next := cloneAgreement(current)
		next.Version = req.Version
		applyAgreementInput(next, &req.Input)
		if err = s.validate(txCtx, next, &req.Input); err != nil {
			return err
		}
		if updated, err = s.repo.Update(txCtx, next); err != nil {
			return err
		}
		return s.syncFactorRecourse(txCtx, updated)
	})
	if err != nil {
		return nil, err
	}

	s.rec.log(actor, &auditEntry{
		resource: permission.ResourceFactoringAgreement,
		entity:   updated,
		previous: previous,
		op:       permission.OpUpdate,
		comment:  "Factoring agreement updated",
	})
	s.rec.publish(ctx, permission.ResourceFactoringAgreement, updated, actor, "updated")
	s.requestFactor(ctx, req.TenantInfo, updated.FactoringCompanyID)
	return updated, nil
}

func (s *AgreementService) Activate(
	ctx context.Context,
	req *factoringports.ChangeAgreementStatusRequest,
	actor *permissionports.RequestActor,
) (*factoring.Agreement, error) {
	return s.transition(ctx, req, actor, &transition{
		label:   "Activating a factoring agreement",
		comment: "Factoring agreement activated",
		apply: func(a *factoring.Agreement, at int64) error {
			if a.Status != factoring.AgreementStatusDraft {
				return errortypes.NewBusinessError("Only a draft agreement can be activated")
			}
			if a.EffectiveTo != nil && *a.EffectiveTo <= at {
				return errortypes.NewValidationError(
					"effectiveTo",
					errortypes.ErrInvalid,
					"The agreement has already ended; change its end date before activating it",
				)
			}
			a.Status = factoring.AgreementStatusActive
			a.ActivatedAt = &at
			return nil
		},
	})
}

func (s *AgreementService) End(
	ctx context.Context,
	req *factoringports.ChangeAgreementStatusRequest,
	actor *permissionports.RequestActor,
) (*factoring.Agreement, error) {
	return s.transition(ctx, req, actor, &transition{
		label:   "Ending a factoring agreement",
		comment: "Factoring agreement ended",
		apply: func(a *factoring.Agreement, at int64) error {
			if a.Status != factoring.AgreementStatusActive {
				return errortypes.NewBusinessError("Only an active agreement can be ended")
			}
			endAt := at
			if req.EndAt != nil {
				endAt = *req.EndAt
			}
			if endAt <= a.EffectiveFrom {
				return errortypes.NewValidationError(
					"endAt",
					errortypes.ErrInvalid,
					"The agreement must end after it took effect",
				)
			}
			if a.EffectiveTo == nil || endAt < *a.EffectiveTo {
				a.EffectiveTo = &endAt
			}
			a.Status = factoring.AgreementStatusEnded
			a.EndedAt = &at
			return nil
		},
	})
}

type transition struct {
	label   string
	comment string
	apply   func(a *factoring.Agreement, at int64) error
}

func (s *AgreementService) transition(
	ctx context.Context,
	req *factoringports.ChangeAgreementStatusRequest,
	actor *permissionports.RequestActor,
	t *transition,
) (*factoring.Agreement, error) {
	if err := requireRequest(req != nil); err != nil {
		return nil, err
	}
	if err := requireActor(actor, t.label); err != nil {
		return nil, err
	}

	var previous, updated *factoring.Agreement
	err := s.db.WithTx(ctx, kernelports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		current, err := s.repo.GetByID(txCtx, factoringports.GetAgreementRequest{
			ID:         req.ID,
			TenantInfo: req.TenantInfo,
		})
		if err != nil {
			return err
		}
		previous = cloneAgreement(current)

		next := cloneAgreement(current)
		next.Version = req.Version
		if err = t.apply(next, s.now()); err != nil {
			return err
		}
		if updated, err = s.repo.Update(txCtx, next); err != nil {
			return err
		}
		return s.syncFactorRecourse(txCtx, updated)
	})
	if err != nil {
		return nil, err
	}

	s.rec.log(actor, &auditEntry{
		resource: permission.ResourceFactoringAgreement,
		entity:   updated,
		previous: previous,
		op:       permission.OpUpdate,
		comment:  t.comment,
	})
	s.rec.publish(ctx, permission.ResourceFactoringAgreement, updated, actor, "updated")
	s.requestFactor(ctx, req.TenantInfo, updated.FactoringCompanyID)
	return updated, nil
}

func (s *AgreementService) validate(
	ctx context.Context,
	entity *factoring.Agreement,
	input *factoringports.AgreementInput,
) error {
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if len(input.DocumentTypeIDs) > maxAgreementDocumentTypes {
		multiErr.Add(
			"documentTypeIds",
			errortypes.ErrInvalid,
			"An agreement can require at most 20 document types",
		)
	}
	if err := validationOf(multiErr); err != nil {
		return err
	}

	factor, err := s.factors.GetByID(ctx, factoringcompanyports.GetFactoringCompanyByIDRequest{
		ID: entity.FactoringCompanyID,
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
	})
	if err != nil {
		return err
	}
	if factor == nil {
		return errortypes.NewValidationError(
			"factoringCompanyId",
			errortypes.ErrInvalid,
			"The factoring company does not exist",
		)
	}
	return nil
}

func (s *AgreementService) syncFactorRecourse(ctx context.Context, a *factoring.Agreement) error {
	if a.Status != factoring.AgreementStatusActive || !a.InForce(s.now()) {
		return nil
	}
	factor, err := s.factors.GetByID(ctx, factoringcompanyports.GetFactoringCompanyByIDRequest{
		ID:         a.FactoringCompanyID,
		TenantInfo: pagination.TenantInfo{OrgID: a.OrganizationID, BuID: a.BusinessUnitID},
	})
	if err != nil {
		return err
	}
	if factor.NonRecourse == a.IsNonRecourse() {
		return nil
	}
	factor.NonRecourse = a.IsNonRecourse()
	_, err = s.factors.Update(ctx, factor)
	return err
}

func (s *AgreementService) requestFactor(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	factorID pulid.ID,
) {
	if s.requester == nil {
		return
	}
	s.requester.RequestEvaluation(ctx, tenantInfo, &factoringports.EvaluationTargets{
		FactoringCompanyIDs: []pulid.ID{factorID},
	})
}

func applyAgreementInput(a *factoring.Agreement, in *factoringports.AgreementInput) {
	a.EffectiveFrom = in.EffectiveFrom
	a.EffectiveTo = in.EffectiveTo
	a.AdvanceRateBP = in.AdvanceRateBP
	a.ReserveRateBP = in.ReserveRateBP
	a.MinimumFeeMinor = in.MinimumFeeMinor
	a.ChargebackWindowDays = in.ChargebackWindowDays
	a.Recourse = in.Recourse
	a.AccountingTreatment = in.AccountingTreatment
	a.FundingDays = in.FundingDays
	a.SubmissionTime = in.SubmissionTime
	a.Notes = in.Notes

	a.FeeTiers = make([]*factoring.FeeTier, 0, len(in.FeeTiers))
	for _, tier := range in.FeeTiers {
		if tier == nil {
			continue
		}
		a.FeeTiers = append(a.FeeTiers, &factoring.FeeTier{
			DaysFrom:  tier.DaysFrom,
			DaysTo:    tier.DaysTo,
			RateBP:    tier.RateBP,
			FlatMinor: tier.FlatMinor,
		})
	}

	seen := make(map[pulid.ID]struct{}, len(in.DocumentTypeIDs))
	a.DocumentTypes = make([]*documenttype.DocumentType, 0, len(in.DocumentTypeIDs))
	for _, id := range in.DocumentTypeIDs {
		if id.IsNil() {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		a.DocumentTypes = append(a.DocumentTypes, &documenttype.DocumentType{ID: id})
	}
}

func cloneAgreement(a *factoring.Agreement) *factoring.Agreement {
	out := *a
	out.FeeTiers = make([]*factoring.FeeTier, 0, len(a.FeeTiers))
	for _, tier := range a.FeeTiers {
		if tier == nil {
			continue
		}
		copied := *tier
		out.FeeTiers = append(out.FeeTiers, &copied)
	}
	out.DocumentTypes = append([]*documenttype.DocumentType(nil), a.DocumentTypes...)
	return &out
}
