package aiauditservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/aiaudit"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

// PersonFilterField is the trail filter for "everything done for or decided
// by this person", which no single column answers.
const PersonFilterField = "personId"

// Service is the trail as the API reads it, verifies it and exports it.
type Service struct {
	ledger    repositories.AIAuditRepository
	projector *Projector
	verifier  *Verifier
	exports   *Exports
	workflows serviceports.WorkflowStarter
	l         *zap.Logger
}

var _ serviceports.AIAuditService = (*Service)(nil)

type ServiceParams struct {
	Ledger    repositories.AIAuditRepository
	Projector *Projector
	Verifier  *Verifier
	Exports   *Exports
	Workflows serviceports.WorkflowStarter
	Logger    *zap.Logger
}

func NewService(p ServiceParams) *Service {
	return &Service{
		ledger:    p.Ledger,
		projector: p.Projector,
		verifier:  p.Verifier,
		exports:   p.Exports,
		workflows: p.Workflows,
		l:         p.Logger.Named("aiaudit.service"),
	}
}

func (s *Service) ListEvents(
	ctx context.Context,
	req *serviceports.ListAIAuditEventsRequest,
) (*pagination.CursorListResult[*aiaudit.AIAuditEvent], error) {
	if req.Filter == nil {
		return nil, errortypes.NewValidationError(
			"filter",
			errortypes.ErrRequired,
			"A filter is required",
		)
	}
	req.Filter.FieldFilters, req.Filter.FilterGroups = NormalizePersonFilters(
		req.Filter.FieldFilters,
		req.Filter.FilterGroups,
	)

	return s.ledger.ListConnection(ctx, &repositories.ListAIAuditEventsRequest{
		Filter:  req.Filter,
		Cursor:  req.Cursor,
		Columns: req.Columns,
	})
}

// NormalizePersonFilters rewrites each "personId" filter into a group that
// matches the person either as the one the work was for or as the one who
// decided it.
func NormalizePersonFilters(
	filters []domaintypes.FieldFilter,
	groups []domaintypes.FilterGroup,
) ([]domaintypes.FieldFilter, []domaintypes.FilterGroup) {
	kept := make([]domaintypes.FieldFilter, 0, len(filters))
	for _, filter := range filters {
		if filter.Field != PersonFilterField {
			kept = append(kept, filter)

			continue
		}

		operator := filter.Operator
		if operator != dbtype.OpIn {
			operator = dbtype.OpEqual
		}
		groups = append(groups, domaintypes.FilterGroup{Filters: []domaintypes.FieldFilter{
			buncolgen.AIAuditEventFilter.OnBehalfOfUserID(operator, filter.Value),
			buncolgen.AIAuditEventFilter.DecidedByUserID(operator, filter.Value),
		}})
	}

	return kept, groups
}

func (s *Service) GetEvent(
	ctx context.Context,
	req repositories.GetAIAuditEventRequest,
) (*aiaudit.AIAuditEvent, error) {
	return s.ledger.GetByID(ctx, req)
}

func (s *Service) ReaderArguments(
	ctx context.Context,
	event *aiaudit.AIAuditEvent,
	ceilings serviceports.FieldCeilings,
) map[string]any {
	return ReaderArguments(ctx, event, ceilings)
}

func (s *Service) ChainStatus(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*serviceports.AIAuditChainStatus, error) {
	return s.verifier.ChainStatus(ctx, tenantInfo)
}

// RequestVerification starts a check of the reader's tenant's chain. A check
// already running is joined rather than started twice.
func (s *Service) RequestVerification(
	ctx context.Context,
	actor *serviceports.RequestActor,
) (*serviceports.AIAuditChainStatus, error) {
	if actor == nil {
		return nil, errortypes.NewAuthenticationError("Authentication required")
	}
	tenantInfo := pagination.TenantInfo{OrgID: actor.OrganizationID, BuID: actor.BusinessUnitID}

	if _, err := s.workflows.StartWorkflow(ctx,
		client.StartWorkflowOptions{
			ID: fmt.Sprintf("ai-audit-verify/%s/%s",
				actor.OrganizationID, actor.BusinessUnitID),
			TaskQueue: temporaltype.AuditTaskQueue,
		},
		serviceports.AIAuditVerifyWorkflowName,
		&serviceports.AIAuditVerifyPayload{
			OrganizationID: actor.OrganizationID,
			BusinessUnitID: actor.BusinessUnitID,
			RequestedBy:    actor.UserID,
		},
	); err != nil {
		s.l.Error("failed to start an AI audit verification", zap.Error(err))

		return nil, errortypes.NewBusinessError(
			"The verification could not be started — try again shortly",
		)
	}

	status, err := s.verifier.ChainStatus(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	status.Verifying = true

	return status, nil
}

func (s *Service) RequestExport(
	ctx context.Context,
	req *serviceports.RequestAIAuditExportRequest,
) (*aiaudit.AIAuditExport, error) {
	if req.Filter != nil {
		req.Filter.FieldFilters, req.Filter.FilterGroups = NormalizePersonFilters(
			req.Filter.FieldFilters,
			req.Filter.FilterGroups,
		)
	}

	return s.exports.Request(ctx, req)
}

func (s *Service) GetExport(
	ctx context.Context,
	req repositories.GetAIAuditExportRequest,
) (*aiaudit.AIAuditExport, error) {
	return s.exports.exports.GetByID(ctx, req)
}

func (s *Service) ListExports(
	ctx context.Context,
	req *repositories.ListAIAuditExportsRequest,
) (*pagination.CursorListResult[*aiaudit.AIAuditExport], error) {
	return s.exports.exports.ListConnection(ctx, req)
}

func (s *Service) ExportDownload(
	ctx context.Context,
	req *serviceports.GetAIAuditExportDownloadRequest,
) (*serviceports.AIAuditExportDownload, error) {
	return s.exports.Download(ctx, req)
}

func (s *Service) SourcePruneHorizon(ctx context.Context, source aiaudit.Source) (int64, error) {
	return s.projector.SourcePruneHorizon(ctx, source)
}
