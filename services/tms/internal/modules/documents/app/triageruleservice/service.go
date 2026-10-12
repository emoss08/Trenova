package triageruleservice

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/documents/app/triagesupport"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/triageports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	maxPreviewCandidates = 5000
	maxPreviewMatches    = 50
	maxSenderDomains     = 20
	maxMailboxes         = 100
	previewTTL           = 2 * time.Minute
	suggestionTTL        = 5 * time.Minute
)

type Params struct {
	fx.In

	Logger        *zap.Logger
	Rules         triageports.TriageRuleRepository
	Insights      triageports.TriageInsightRepository
	Mailboxes     inboundmessageports.InboundMailboxRepository
	Organizations tenantports.OrganizationRepository
	Permissions   permissionports.PermissionEngine
	Activities    inboundmessageports.TriageActivityRecorder
	Cache         triageports.TriageCache `optional:"true"`
	Audit         auditports.AuditService `optional:"true"`
}

type Service struct {
	l           *zap.Logger
	rules       triageports.TriageRuleRepository
	insights    triageports.TriageInsightRepository
	mailboxes   inboundmessageports.InboundMailboxRepository
	permissions permissionports.PermissionEngine
	activities  inboundmessageports.TriageActivityRecorder
	cache       triageports.TriageCache
	audit       auditports.AuditService
	clock       *triagesupport.Clock
}

func New(p Params) *Service {
	return &Service{
		l:           p.Logger.Named("service.triage-rule"),
		rules:       p.Rules,
		insights:    p.Insights,
		mailboxes:   p.Mailboxes,
		permissions: p.Permissions,
		activities:  p.Activities,
		cache:       p.Cache,
		audit:       p.Audit,
		clock:       triagesupport.NewClock(p.Organizations, nil),
	}
}

var (
	_ inboundmessageports.TriageRuleTeacher = (*Service)(nil)
	_ triageports.TriageRuleStats           = (*Service)(nil)
)

func mayReadInbox(
	ctx context.Context,
	engine permissionports.PermissionEngine,
	tenantInfo pagination.TenantInfo,
	userID pulid.ID,
) (bool, error) {
	result, err := engine.Check(ctx, &permissionports.PermissionCheckRequest{
		PrincipalType:  permissionports.PrincipalTypeUser,
		PrincipalID:    userID,
		UserID:         userID,
		BusinessUnitID: tenantInfo.BuID,
		OrganizationID: tenantInfo.OrgID,
		Resource:       permission.ResourceInboundMessage.String(),
		Operation:      permission.OpRead,
	})
	if err != nil {
		return false, err
	}

	return result.Allowed, nil
}

func enabledRules(
	ctx context.Context,
	deps ruleSource,
	tenantInfo pagination.TenantInfo,
) ([]*triage.Rule, error) {
	if deps.cache != nil {
		cached, hit, err := deps.cache.EnabledRules(ctx, tenantInfo)
		if err != nil {
			deps.l.Warn("triage rule cache unavailable", zap.Error(err))
		}
		if hit {
			return cached, nil
		}
	}

	rules, err := deps.rules.List(ctx, triageports.ListRulesRequest{
		TenantInfo:  tenantInfo,
		EnabledOnly: true,
	})
	if err != nil {
		return nil, err
	}

	if deps.cache != nil {
		if err = deps.cache.SetEnabledRules(ctx, tenantInfo, rules); err != nil {
			deps.l.Warn("could not cache the enabled triage rules", zap.Error(err))
		}
	}

	return rules, nil
}

type ruleSource struct {
	l     *zap.Logger
	rules triageports.TriageRuleRepository
	cache triageports.TriageCache
}

func (s *Service) source() ruleSource {
	return ruleSource{l: s.l, rules: s.rules, cache: s.cache}
}

func (s *Service) rulesChanged(ctx context.Context, tenantInfo pagination.TenantInfo) {
	if s.cache == nil {
		return
	}
	if err := s.cache.DropEnabledRules(ctx, tenantInfo); err != nil {
		s.l.Warn("could not drop the cached triage rules",
			zap.String("organizationId", tenantInfo.OrgID.String()), zap.Error(err))
	}
	triagesupport.Invalidate(ctx, s.cache, s.l, tenantInfo)
}
