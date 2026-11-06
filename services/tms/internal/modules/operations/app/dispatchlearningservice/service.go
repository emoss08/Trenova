package dispatchlearningservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatcheconomics"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchlearning"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchcontrolports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchlearningports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	decisionBatch     = 200
	batchesPerRun     = 10
	defaultTenantPage = 100
)

type Params struct {
	fx.In

	Logger       *zap.Logger
	Repo         dispatchlearningports.Repository
	Controls     dispatchcontrolports.DispatchControlRepository
	AuditService auditports.AuditService
	Clock        timeutils.Clock
}

type Service struct {
	l        *zap.Logger
	repo     dispatchlearningports.Repository
	controls dispatchcontrolports.DispatchControlRepository
	audit    auditports.AuditService
	clock    timeutils.Clock
}

func New(p Params) *Service {
	return &Service{
		l:        p.Logger.Named("service.dispatch-learning"),
		repo:     p.Repo,
		controls: p.Controls,
		audit:    p.AuditService,
		clock:    p.Clock,
	}
}

func (s *Service) Multipliers(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (map[dispatcheconomics.Component]float64, error) {
	style, err := s.repo.Get(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	if style == nil {
		return map[dispatcheconomics.Component]float64{}, nil
	}

	return style.Multipliers(), nil
}

func (s *Service) StartLearning(ctx context.Context, tenantInfo pagination.TenantInfo) error {
	now := s.clock()
	style, err := s.repo.GetOrCreate(
		ctx,
		dispatchlearning.New(tenantInfo.OrgID, tenantInfo.BuID, now),
	)
	if err != nil {
		return err
	}
	if now <= style.ObservedThroughAt {
		return nil
	}
	style.StartObserving(now)
	_, err = s.repo.Save(ctx, style)

	return err
}

func (s *Service) Get(
	ctx context.Context,
	req *dispatchlearningports.GetStyleRequest,
) (*dispatchlearningports.Style, error) {
	control, err := s.controls.GetOrCreate(ctx, req.TenantInfo.OrgID, req.TenantInfo.BuID)
	if err != nil {
		return nil, err
	}
	style, err := s.repo.Get(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}

	return ViewOf(style, control.LearnDispatchStyle), nil
}

func (s *Service) Reset(
	ctx context.Context,
	req *dispatchlearningports.ResetStyleRequest,
) (*dispatchlearningports.Style, error) {
	control, err := s.controls.GetOrCreate(ctx, req.TenantInfo.OrgID, req.TenantInfo.BuID)
	if err != nil {
		return nil, err
	}
	style, err := s.repo.Get(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	if style == nil {
		return ViewOf(nil, control.LearnDispatchStyle), nil
	}

	original := *style
	style.Reset(s.clock())
	if err = validate(style); err != nil {
		return nil, err
	}

	saved, err := s.repo.Save(ctx, style)
	if err != nil {
		return nil, err
	}

	if err = s.audit.LogAction(
		&auditports.LogActionParams{
			Resource:       permission.ResourceDispatchControl,
			ResourceID:     saved.ID.String(),
			Operation:      permission.OpUpdate,
			UserID:         req.Actor.UserID,
			PrincipalType:  req.Actor.PrincipalType,
			PrincipalID:    req.Actor.PrincipalID,
			APIKeyID:       req.Actor.APIKeyID,
			CurrentState:   jsonutils.MustToJSON(saved),
			PreviousState:  jsonutils.MustToJSON(&original),
			OrganizationID: saved.OrganizationID,
			BusinessUnitID: saved.BusinessUnitID,
		},
		auditservice.WithComment("Learned dispatch style reset"),
		auditservice.WithDiff(&original, saved),
		auditservice.WithCritical(),
	); err != nil {
		s.l.Error("failed to log dispatch style reset", zap.Error(err))
	}

	return ViewOf(saved, control.LearnDispatchStyle), nil
}

func (s *Service) Learn(
	ctx context.Context,
	req *dispatchlearningports.LearnRequest,
) (*dispatchlearningports.LearnResult, error) {
	result := &dispatchlearningports.LearnResult{}
	control, err := s.controls.GetOrCreate(ctx, req.TenantInfo.OrgID, req.TenantInfo.BuID)
	if err != nil {
		return nil, err
	}
	if !control.LearnDispatchStyle {
		return result, nil
	}

	style, err := s.repo.GetOrCreate(
		ctx,
		dispatchlearning.New(req.TenantInfo.OrgID, req.TenantInfo.BuID, s.clock()),
	)
	if err != nil {
		return nil, err
	}

	for range batchesPerRun {
		page, listErr := s.repo.ListDecisions(ctx, &dispatchlearningports.ListDecisionsRequest{
			TenantInfo: req.TenantInfo,
			After: dispatchlearningports.DecisionCursor{
				At: style.ObservedThroughAt,
				ID: style.ObservedThroughID,
			},
			Limit: decisionBatch,
		})
		if listErr != nil {
			return nil, listErr
		}
		if page.Scanned == 0 {
			break
		}

		for _, decision := range page.Decisions {
			tally(result, style.Learn(decision))
		}
		style.Advance(page.Last.At, page.Last.ID)
		if err = validate(style); err != nil {
			return nil, err
		}
		if style, err = s.repo.Save(ctx, style); err != nil {
			return nil, err
		}
		result.Scanned += page.Scanned

		if page.Scanned < decisionBatch {
			break
		}
	}

	return result, nil
}

func (s *Service) ListLearningTenants(
	ctx context.Context,
	req *dispatchlearningports.ListTenantsRequest,
) ([]pagination.TenantInfo, error) {
	if req.Limit <= 0 {
		req.Limit = defaultTenantPage
	}

	return s.repo.ListLearningTenants(ctx, req)
}

func ViewOf(style *dispatchlearning.DispatchStyle, enabled bool) *dispatchlearningports.Style {
	view := &dispatchlearningports.Style{
		Enabled: enabled,
		Weights: []dispatchlearning.LearnedWeight{},
	}
	if style == nil {
		return view
	}

	view.Decisions = style.Decisions
	view.Approvals = style.Approvals
	view.Overrides = style.Overrides
	view.Declines = style.Declines
	view.Weights = style.Learned()
	if style.LearningSince > 0 {
		since := style.LearningSince
		view.LearningSince = &since
	}
	if style.Decisions > 0 && style.UpdatedAt > 0 {
		updated := style.UpdatedAt
		view.UpdatedAt = &updated
	}

	return view
}

func validate(style *dispatchlearning.DispatchStyle) error {
	multiErr := errortypes.NewMultiError()
	style.Validate(multiErr)
	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func tally(result *dispatchlearningports.LearnResult, outcome dispatchlearning.Outcome) {
	result.Learned++
	switch outcome {
	case dispatchlearning.OutcomeApproved:
		result.Approvals++
	case dispatchlearning.OutcomeOverridden:
		result.Overrides++
	case dispatchlearning.OutcomeDeclined:
		result.Declines++
	}
}
