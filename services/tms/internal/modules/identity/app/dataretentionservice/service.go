package dataretentionservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/dataretention"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultAuditRetentionDays = 120
)

type Params struct {
	fx.In

	Logger *zap.Logger
	Repo   tenantports.DataRetentionRepository
}

type Service struct {
	l    *zap.Logger
	repo tenantports.DataRetentionRepository
}

func New(p Params) *Service {
	return &Service{
		l:    p.Logger.Named("service.data-retention"),
		repo: p.Repo,
	}
}

func (s *Service) Get(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*dataretention.DataRetention, error) {
	entity, err := s.repo.Get(ctx, tenantports.GetDataRetentionRequest{
		UserID: tenantInfo.UserID,
		OrgID:  tenantInfo.OrgID,
		BuID:   tenantInfo.BuID,
	})
	if err == nil {
		return entity, nil
	}
	if !errortypes.IsNotFoundError(err) {
		return nil, err
	}
	return &dataretention.DataRetention{
		OrganizationID:               tenantInfo.OrgID,
		BusinessUnitID:               tenantInfo.BuID,
		AuditRetentionPeriod:         defaultAuditRetentionDays,
		AIFeedbackRetentionPeriod:    dataretention.DefaultAIFeedbackRetentionDays,
		AgentEvalCaseRetentionPeriod: dataretention.DefaultAgentEvalCaseRetentionDays,
		AIAuditRetentionPeriod:       dataretention.DefaultAIAuditRetentionDays,
		AICorrectionRetentionPeriod:  dataretention.DefaultAICorrectionRetentionDays,
	}, nil
}

type UpdateDataRetentionRequest struct {
	TenantInfo                    pagination.TenantInfo `json:"-"`
	AuditRetentionPeriod          int                   `json:"auditRetentionPeriod"`
	EDIInboundFileRetentionPeriod int                   `json:"ediInboundFileRetentionPeriod"`
	EDIMessageRetentionPeriod     int                   `json:"ediMessageRetentionPeriod"`
	AIFeedbackRetentionPeriod     int                   `json:"aiFeedbackRetentionPeriod"`
	AgentEvalCaseRetentionPeriod  *int                  `json:"agentEvalCaseRetentionPeriod,omitempty"`
	AIAuditRetentionPeriod        *int                  `json:"aiAuditRetentionPeriod,omitempty"`
	AICorrectionRetentionPeriod   *int                  `json:"aiCorrectionRetentionPeriod,omitempty"`
}

func (r *UpdateDataRetentionRequest) keepsCurrent() bool {
	return r.AgentEvalCaseRetentionPeriod == nil || r.AIAuditRetentionPeriod == nil ||
		r.AICorrectionRetentionPeriod == nil
}

func (s *Service) Update(
	ctx context.Context,
	req *UpdateDataRetentionRequest,
) (*dataretention.DataRetention, error) {
	entity := &dataretention.DataRetention{
		OrganizationID:                req.TenantInfo.OrgID,
		BusinessUnitID:                req.TenantInfo.BuID,
		AuditRetentionPeriod:          req.AuditRetentionPeriod,
		EDIInboundFileRetentionPeriod: req.EDIInboundFileRetentionPeriod,
		EDIMessageRetentionPeriod:     req.EDIMessageRetentionPeriod,
		AIFeedbackRetentionPeriod:     req.AIFeedbackRetentionPeriod,
	}
	if req.keepsCurrent() {
		current, err := s.Get(ctx, req.TenantInfo)
		if err != nil {
			return nil, err
		}
		entity.AgentEvalCaseRetentionPeriod = current.AgentEvalCaseRetentionPeriod
		entity.AIAuditRetentionPeriod = current.AIAuditRetentionDays()
		entity.AICorrectionRetentionPeriod = current.AICorrectionRetentionDays()
	}
	if req.AgentEvalCaseRetentionPeriod != nil {
		entity.AgentEvalCaseRetentionPeriod = *req.AgentEvalCaseRetentionPeriod
	}
	if req.AIAuditRetentionPeriod != nil {
		entity.AIAuditRetentionPeriod = *req.AIAuditRetentionPeriod
	}
	if req.AICorrectionRetentionPeriod != nil {
		entity.AICorrectionRetentionPeriod = *req.AICorrectionRetentionPeriod
	}
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}
	return s.repo.Upsert(ctx, entity)
}
