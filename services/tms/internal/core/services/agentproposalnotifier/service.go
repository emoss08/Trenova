// Package agentproposalnotifier brings a waiting proposal to the people who
// can decide it.
//
// A background agent's proposal used to appear in one place: the activity
// table in AI Control. Nobody sits on that page. The dispatcher who could
// approve a reassignment in ten seconds found out about it a day later, when
// the proposal had expired and the load had gone out uncovered. Now every
// person who holds the decision permission is told in the app the moment the
// proposal is stored, and told again, with more weight and by email, when it
// has sat for hours.
package agentproposalnotifier

import (
	"context"
	"fmt"
	"html/template"
	"net/url"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentshadow"
	"github.com/emoss08/trenova/internal/core/services/notificationservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	EventProposalsPending  = "agent.proposals_pending"
	EventProposalsReminder = "agent.proposals_reminder"

	notificationSource = "agentproposalnotifier"
	proposalsPath      = "/desk/decisions"
	maxListedTools     = 3
	secondsPerHour     = 3600
)

type deciderLister interface {
	ListUsersWithPermission(
		ctx context.Context,
		req repositories.ListUsersWithPermissionRequest,
	) ([]repositories.PermittedUser, error)
}

type notifier interface {
	Create(
		ctx context.Context,
		entity *notification.Notification,
	) (*notification.Notification, error)
}

type mailer interface {
	Send(context.Context, *services.SendEmailRequest) (*email.Message, error)
}

type renderer interface {
	RenderMessage(
		ctx context.Context,
		req *services.RenderMessageRequest,
	) (*services.RenderedMessage, error)
}

type shadowReader interface {
	Organization(ctx context.Context, tenantInfo pagination.TenantInfo) (bool, error)
}

type proposalStore interface {
	ListPendingForReminder(
		ctx context.Context,
		req repositories.ListPendingProposalsForReminderRequest,
	) ([]*agent.AgentProposal, error)
	MarkReminded(ctx context.Context, req repositories.MarkProposalsRemindedRequest) (int, error)
}

type runReader interface {
	GetByID(ctx context.Context, req repositories.GetAgentRunByIDRequest) (*agent.AgentRun, error)
}

type definitionReader interface {
	GetByID(
		ctx context.Context,
		req repositories.GetAgentDefinitionByIDRequest,
	) (*agentdefinition.Definition, error)
}

type Params struct {
	fx.In

	Logger      *zap.Logger
	Config      *config.Config
	Roles       repositories.RoleRepository
	Proposals   repositories.AgentProposalRepository
	Runs        repositories.AgentRunRepository
	Definitions repositories.AgentDefinitionRepository
	Orgs        repositories.OrganizationRepository
	Shadow      *agentshadow.Resolver
	// The channels are optional: an in-app notice needs the notification
	// service, an email needs the email service and a template resolver, and
	// the ledger of who was reminded is kept either way.
	Notifications *notificationservice.Service      `optional:"true"`
	Email         services.EmailService             `optional:"true"`
	Templates     services.DocumentTemplateResolver `optional:"true"`
	Inliner       services.AssetInliner             `optional:"true"`
}

type Service struct {
	l             *zap.Logger
	webBaseURL    string
	roles         deciderLister
	proposals     proposalStore
	runs          runReader
	definitions   definitionReader
	orgs          repositories.OrganizationRepository
	shadow        shadowReader
	notifications notifier
	email         mailer
	templates     renderer
	inliner       services.AssetInliner
}

func New(p Params) services.AgentProposalNotifier {
	svc := &Service{
		l:           p.Logger.Named("service.agentproposalnotifier"),
		roles:       p.Roles,
		proposals:   p.Proposals,
		runs:        p.Runs,
		definitions: p.Definitions,
		orgs:        p.Orgs,
		shadow:      p.Shadow,
		inliner:     p.Inliner,
	}
	if p.Config != nil {
		svc.webBaseURL = p.Config.App.GetWebBaseURL()
	}
	if p.Notifications != nil {
		svc.notifications = p.Notifications
	}
	if p.Email != nil {
		svc.email = p.Email
	}
	if p.Templates != nil {
		svc.templates = p.Templates
	}

	return svc
}

// NotifyPending tells the deciders about a run's proposals as soon as they
// are stored. A chat run is skipped: the person who asked is looking at the
// answer. A run in shadow is skipped too, because its proposals are hidden
// and a notice that leads to nothing to decide is worse than none.
func (s *Service) NotifyPending(ctx context.Context, notice services.PendingProposalsNotice) error {
	if notice.Run == nil || notice.Definition == nil || notice.Run.Trigger == agent.RunTriggerChat {
		return nil
	}

	pending := pendingOf(notice.Proposals)
	if len(pending) == 0 {
		return nil
	}

	tenantInfo := pagination.TenantInfo{
		OrgID: notice.Run.OrganizationID,
		BuID:  notice.Run.BusinessUnitID,
	}
	shadow, err := s.inShadow(ctx, tenantInfo, notice.Definition)
	if err != nil || shadow {
		return err
	}

	deciders, err := s.deciders(ctx, tenantInfo, notice.Run.StartedAt)
	if err != nil {
		return err
	}
	if len(deciders) == 0 {
		s.l.Warn("proposals are pending but nobody in the organization can decide them",
			zap.String("organizationId", tenantInfo.OrgID.String()),
			zap.String("run", notice.Run.ID.String()),
		)

		return nil
	}

	group := proposalGroup{
		tenant:     tenantInfo,
		run:        notice.Run,
		definition: notice.Definition,
		proposals:  pending,
	}
	for _, decider := range deciders {
		s.notify(ctx, group, decider, deciderNotice{
			eventType: EventProposalsPending,
			priority:  notification.PriorityMedium,
			title: fmt.Sprintf("%s has %s waiting for a decision",
				notice.Definition.Name, countChanges(len(pending))),
			message: fmt.Sprintf("It proposed %s. Nothing runs until someone decides.",
				describeTools(pending)),
		})
	}

	return nil
}

// RemindPending finds proposals from background runs that have waited longer
// than the cut-off and tells their deciders again, in the app at high
// priority and by email. Each proposal is reminded about once; the mark is
// written whether or not a channel delivered, so a broken mailer costs one
// reminder rather than one every sweep.
func (s *Service) RemindPending(
	ctx context.Context,
	req services.RemindPendingProposalsRequest,
) (int, error) {
	proposals, err := s.proposals.ListPendingForReminder(
		ctx,
		repositories.ListPendingProposalsForReminderRequest{
			Before: req.Now - int64(req.OlderThan.Seconds()),
			Now:    req.Now,
			Limit:  req.Limit,
		},
	)
	if err != nil {
		return 0, fmt.Errorf("list proposals for reminder: %w", err)
	}
	if len(proposals) == 0 {
		return 0, nil
	}

	reminded := 0
	for _, group := range groupByRun(proposals) {
		if err = s.remindGroup(ctx, group, req.Now); err != nil {
			s.l.Error("failed to remind about pending proposals",
				zap.String("run", group.runID.String()),
				zap.Error(err),
			)

			continue
		}
		reminded += len(group.proposals)
	}

	return reminded, nil
}

type proposalGroup struct {
	tenant     pagination.TenantInfo
	runID      pulid.ID
	run        *agent.AgentRun
	definition *agentdefinition.Definition
	proposals  []*agent.AgentProposal
}

func (s *Service) remindGroup(ctx context.Context, group proposalGroup, now int64) error {
	run, err := s.runs.GetByID(ctx, repositories.GetAgentRunByIDRequest{
		ID:         group.runID,
		TenantInfo: &group.tenant,
	})
	if err != nil {
		return fmt.Errorf("read run: %w", err)
	}
	group.run = run

	if run.AgentDefinitionID.IsNotNil() {
		definition, defErr := s.definitions.GetByID(ctx, repositories.GetAgentDefinitionByIDRequest{
			ID:         run.AgentDefinitionID,
			TenantInfo: group.tenant,
		})
		if defErr != nil {
			return fmt.Errorf("read agent: %w", defErr)
		}
		group.definition = definition
	}

	// Marked first: a proposal in shadow is hidden, so there is nothing to
	// remind anyone of, and it must not be picked up again by the next sweep.
	if _, err = s.proposals.MarkReminded(ctx, repositories.MarkProposalsRemindedRequest{
		IDs: ids(group.proposals),
		At:  now,
	}); err != nil {
		return fmt.Errorf("mark reminded: %w", err)
	}

	shadow, err := s.inShadow(ctx, group.tenant, group.definition)
	if err != nil || shadow {
		return err
	}

	deciders, err := s.deciders(ctx, group.tenant, now)
	if err != nil {
		return err
	}

	waitingHours := int((now - oldest(group.proposals)) / secondsPerHour)
	agentName := agentNameOf(group.definition)
	for _, decider := range deciders {
		s.notify(ctx, group, decider, deciderNotice{
			eventType: EventProposalsReminder,
			priority:  notification.PriorityHigh,
			title: fmt.Sprintf("%s still has %s waiting after %d hours",
				agentName, countChanges(len(group.proposals)), waitingHours),
			message: fmt.Sprintf(
				"It proposed %s. Nobody has decided, and nothing runs until someone does.",
				describeTools(group.proposals),
			),
		})
		s.mail(ctx, group, decider, waitingHours)
	}

	return nil
}

func (s *Service) inShadow(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	definition *agentdefinition.Definition,
) (bool, error) {
	if s.shadow == nil {
		return false, nil
	}

	organizationShadow, err := s.shadow.Organization(ctx, tenantInfo)
	if err != nil {
		return false, fmt.Errorf("read organization shadow: %w", err)
	}
	if definition == nil {
		return organizationShadow, nil
	}

	return definition.EffectiveShadow(organizationShadow), nil
}

func (s *Service) deciders(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	now int64,
) ([]repositories.PermittedUser, error) {
	users, err := s.roles.ListUsersWithPermission(ctx, repositories.ListUsersWithPermissionRequest{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		Resource:       permission.ResourceAgentProposal,
		Operation:      permission.OpUpdate,
		Now:            now,
	})
	if err != nil {
		return nil, fmt.Errorf("list deciders: %w", err)
	}

	return users, nil
}

type deciderNotice struct {
	eventType string
	priority  notification.Priority
	title     string
	message   string
}

func (s *Service) notify(
	ctx context.Context,
	group proposalGroup,
	decider repositories.PermittedUser,
	n deciderNotice,
) {
	if s.notifications == nil {
		return
	}

	buID := group.tenant.BuID
	userID := decider.UserID
	correlation := group.runID.String()
	if group.run != nil {
		correlation = group.run.ID.String()
	}

	if _, err := s.notifications.Create(ctx, &notification.Notification{
		OrganizationID: group.tenant.OrgID,
		BusinessUnitID: &buID,
		TargetUserID:   &userID,
		Channel:        notification.ChannelUser,
		EventType:      n.eventType,
		Priority:       n.priority,
		Title:          n.title,
		Message:        n.message,
		Source:         notificationSource,
		CorrelationID:  &correlation,
		Data: map[string]any{
			"link":        reviewPath(correlation),
			"runId":       correlation,
			"agentId":     agentIDOf(group.definition),
			"agentName":   agentNameOf(group.definition),
			"count":       len(group.proposals),
			"proposalIds": idStrings(group.proposals),
		},
		RelatedEntities: map[string]any{
			"agentRunId": correlation,
		},
	}); err != nil {
		s.l.Warn("failed to create pending proposal notification",
			zap.String("run", correlation),
			zap.String("userId", decider.UserID.String()),
			zap.Error(err),
		)
	}
}

func (s *Service) mail(
	ctx context.Context,
	group proposalGroup,
	decider repositories.PermittedUser,
	waitingHours int,
) {
	address := strings.TrimSpace(decider.EmailAddress)
	if s.email == nil || s.templates == nil || s.webBaseURL == "" || address == "" {
		return
	}

	runID := group.runID.String()
	data := &documenttemplate.AgentProposalReminderContext{
		RecipientFirstName: stringutils.FirstName(decider.Name),
		AgentName:          agentNameOf(group.definition),
		PendingCount:       len(group.proposals),
		WaitingHours:       waitingHours,
		Proposals:          reminderLines(group.proposals),
		//nolint:gosec // configured base URL + server-encoded path
		ReviewURL: template.URL(s.webBaseURL + reviewPath(runID)),
	}
	s.brand(ctx, group.tenant, data)

	locale, _ := i18n.Parse(decider.Locale)
	rendered, err := s.templates.RenderMessage(ctx, &services.RenderMessageRequest{
		TenantInfo:        group.tenant,
		Kind:              documenttemplate.KindAgentProposalReminderEmail,
		Data:              data,
		ReferenceID:       group.runID,
		UserID:            decider.UserID,
		FallbackToBuiltIn: true,
		Locale:            locale,
	})
	if err != nil {
		s.l.Warn(
			"failed to render proposal reminder email",
			zap.String("run", runID),
			zap.Error(err),
		)

		return
	}

	if _, err = s.email.Send(ctx, &services.SendEmailRequest{
		TenantInfo:     group.tenant,
		Purpose:        email.PurposeNotifications,
		To:             []string{address},
		Subject:        rendered.Subject,
		HTML:           rendered.HTML,
		Text:           rendered.Text,
		IdempotencyKey: "agent-proposal-reminder-" + runID + "-" + decider.UserID.String(),
	}); err != nil {
		s.l.Warn("failed to queue proposal reminder email",
			zap.String("run", runID),
			zap.String("userId", decider.UserID.String()),
			zap.Error(err),
		)
	}
}

func (s *Service) brand(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	out *documenttemplate.AgentProposalReminderContext,
) {
	if s.orgs == nil {
		return
	}

	org, err := s.orgs.GetByID(ctx, repositories.GetOrganizationByIDRequest{TenantInfo: tenantInfo})
	if err != nil || org == nil {
		return
	}

	out.CompanyName = org.Name
	if dataURI, logoErr := services.ResolveLogoDataURI(ctx, s.inliner, org.LogoURL); logoErr == nil {
		out.LogoDataURI = dataURI
	}
}

// reviewPath opens the Desk's decisions queue on the run, so the person
// lands on exactly the rows they were told about.
func reviewPath(runID string) string {
	query := url.Values{}
	query.Set("run", runID)

	return proposalsPath + "?" + query.Encode()
}

func groupByRun(proposals []*agent.AgentProposal) []proposalGroup {
	index := make(map[pulid.ID]int, len(proposals))
	groups := make([]proposalGroup, 0, len(proposals))
	for _, proposal := range proposals {
		at, ok := index[proposal.RunID]
		if !ok {
			at = len(groups)
			index[proposal.RunID] = at
			groups = append(groups, proposalGroup{
				tenant: pagination.TenantInfo{
					OrgID: proposal.OrganizationID,
					BuID:  proposal.BusinessUnitID,
				},
				runID: proposal.RunID,
			})
		}
		groups[at].proposals = append(groups[at].proposals, proposal)
	}

	return groups
}

func pendingOf(proposals []*agent.AgentProposal) []*agent.AgentProposal {
	pending := make([]*agent.AgentProposal, 0, len(proposals))
	for _, proposal := range proposals {
		if proposal != nil && proposal.Status == agent.ProposalStatusPending {
			pending = append(pending, proposal)
		}
	}

	return pending
}

func oldest(proposals []*agent.AgentProposal) int64 {
	var at int64
	for _, proposal := range proposals {
		if at == 0 || proposal.CreatedAt < at {
			at = proposal.CreatedAt
		}
	}

	return at
}

func ids(proposals []*agent.AgentProposal) []pulid.ID {
	out := make([]pulid.ID, 0, len(proposals))
	for _, proposal := range proposals {
		out = append(out, proposal.ID)
	}

	return out
}

func idStrings(proposals []*agent.AgentProposal) []string {
	out := make([]string, 0, len(proposals))
	for _, proposal := range proposals {
		out = append(out, proposal.ID.String())
	}

	return out
}

func reminderLines(proposals []*agent.AgentProposal) []documenttemplate.ProposalReminderLine {
	lines := make([]documenttemplate.ProposalReminderLine, 0, len(proposals))
	for _, proposal := range proposals {
		lines = append(lines, documenttemplate.ProposalReminderLine{
			Tool:      stringutils.HumanizeSnakeCase(proposal.ToolName),
			Rationale: proposal.Rationale,
		})
	}

	return lines
}

// describeTools names the first few tools and counts the rest, so a notice
// says what was proposed without becoming the list it links to.
func describeTools(proposals []*agent.AgentProposal) string {
	names := make([]string, 0, maxListedTools)
	seen := make(map[string]struct{}, len(proposals))
	for _, proposal := range proposals {
		if _, dup := seen[proposal.ToolName]; dup {
			continue
		}
		seen[proposal.ToolName] = struct{}{}
		if len(names) < maxListedTools {
			names = append(names, stringutils.HumanizeSnakeCase(proposal.ToolName))
		}
	}

	rest := len(seen) - len(names)
	switch {
	case rest > 0:
		return strings.Join(names, ", ") + " and " + strconv.Itoa(rest) + " more"
	case len(names) > 1:
		return strings.Join(names[:len(names)-1], ", ") + " and " + names[len(names)-1]
	default:
		return strings.Join(names, "")
	}
}

func countChanges(n int) string {
	if n == 1 {
		return "a change"
	}

	return strconv.Itoa(n) + " changes"
}

func agentNameOf(definition *agentdefinition.Definition) string {
	if definition == nil {
		return "An agent"
	}

	return definition.Name
}

func agentIDOf(definition *agentdefinition.Definition) string {
	if definition == nil {
		return ""
	}

	return definition.ID.String()
}
