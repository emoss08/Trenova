package dataretentionservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/dataretention"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger  *zap.Logger
	Repo    tenantports.DataRetentionRepository
	Auditor permissionports.SecurityAuditor
}

type Service struct {
	l       *zap.Logger
	repo    tenantports.DataRetentionRepository
	auditor permissionports.SecurityAuditor
}

func New(p Params) *Service {
	return &Service{
		l:       p.Logger.Named("service.data-retention"),
		repo:    p.Repo,
		auditor: p.Auditor,
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
		OrganizationID:                tenantInfo.OrgID,
		BusinessUnitID:                tenantInfo.BuID,
		AuditAccessRetentionDays:      dataretention.MinAuditAccessRetentionDays,
		AuditOperationalRetentionDays: dataretention.MinAuditOperationalRetentionDays,
		AuditFinancialRetentionDays:   dataretention.MinAuditFinancialRetentionDays,
		AuditSecurityRetentionDays:    dataretention.MinAuditSecurityRetentionDays,
		AIFeedbackRetentionPeriod:     dataretention.DefaultAIFeedbackRetentionDays,
		AgentEvalCaseRetentionPeriod:  dataretention.DefaultAgentEvalCaseRetentionDays,
		AIAuditRetentionPeriod:        dataretention.DefaultAIAuditRetentionDays,
		AICorrectionRetentionPeriod:   dataretention.DefaultAICorrectionRetentionDays,
	}, nil
}

type UpdateDataRetentionRequest struct {
	TenantInfo                    pagination.TenantInfo  `json:"-"`
	Actor                         kernelports.AuditActor `json:"-"`
	AuditAccessRetentionDays      int                    `json:"auditAccessRetentionDays"`
	AuditOperationalRetentionDays int                    `json:"auditOperationalRetentionDays"`
	AuditFinancialRetentionDays   int                    `json:"auditFinancialRetentionDays"`
	AuditSecurityRetentionDays    int                    `json:"auditSecurityRetentionDays"`
	EDIInboundFileRetentionPeriod int                    `json:"ediInboundFileRetentionPeriod"`
	EDIMessageRetentionPeriod     int                    `json:"ediMessageRetentionPeriod"`
	AIFeedbackRetentionPeriod     int                    `json:"aiFeedbackRetentionPeriod"`
	AgentEvalCaseRetentionPeriod  *int                   `json:"agentEvalCaseRetentionPeriod,omitempty"`
	AIAuditRetentionPeriod        *int                   `json:"aiAuditRetentionPeriod,omitempty"`
	AICorrectionRetentionPeriod   *int                   `json:"aiCorrectionRetentionPeriod,omitempty"`
}

func (r *UpdateDataRetentionRequest) keepsCurrent() bool {
	return r.AgentEvalCaseRetentionPeriod == nil || r.AIAuditRetentionPeriod == nil ||
		r.AICorrectionRetentionPeriod == nil
}

func (s *Service) Update(
	ctx context.Context,
	req *UpdateDataRetentionRequest,
) (*dataretention.DataRetention, error) {
	current, err := s.Get(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	entity := &dataretention.DataRetention{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		AuditAccessRetentionDays: orCurrent(
			req.AuditAccessRetentionDays,
			current.AuditRetentionDays("Access"),
		),
		AuditOperationalRetentionDays: orCurrent(
			req.AuditOperationalRetentionDays,
			current.AuditRetentionDays("Operational"),
		),
		AuditFinancialRetentionDays: orCurrent(
			req.AuditFinancialRetentionDays,
			current.AuditRetentionDays("Financial"),
		),
		AuditSecurityRetentionDays: orCurrent(
			req.AuditSecurityRetentionDays,
			current.AuditRetentionDays("Security"),
		),
		EDIInboundFileRetentionPeriod: req.EDIInboundFileRetentionPeriod,
		EDIMessageRetentionPeriod:     req.EDIMessageRetentionPeriod,
		AIFeedbackRetentionPeriod:     req.AIFeedbackRetentionPeriod,
	}
	if req.keepsCurrent() {
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
	updated, err := s.repo.Upsert(ctx, entity)
	if err != nil {
		return nil, err
	}
	if s.auditor != nil {
		s.auditor.RecordChange(ctx, &permissionports.SecurityChange{
			Resource:       permission.ResourceOrganization,
			ResourceID:     req.TenantInfo.OrgID.String(),
			Operation:      permission.OpUpdate,
			Action:         "data_retention.updated",
			Actor:          req.Actor,
			OrganizationID: req.TenantInfo.OrgID,
			BusinessUnitID: req.TenantInfo.BuID,
			Before:         current,
			After:          updated,
			Comment:        "Data retention changed",
		})
	}
	return updated, nil
}

func orCurrent(requested, current int) int {
	if requested == 0 {
		return current
	}
	return requested
}
