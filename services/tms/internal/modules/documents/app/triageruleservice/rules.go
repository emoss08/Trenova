package triageruleservice

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/triageports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const (
	duplicateSentence  = "Nova already has this rule"
	assigneeCannotRead = "That person cannot read the triage queue"
)

type SaveRequest struct {
	TenantInfo pagination.TenantInfo
	ActorID    pulid.ID
	Rule       *triage.Rule
}

type UpdateRequest struct {
	TenantInfo pagination.TenantInfo
	ActorID    pulid.ID
	ID         pulid.ID
	Version    int64
	Rule       *triage.Rule
}

type SetEnabledRequest struct {
	TenantInfo pagination.TenantInfo
	ActorID    pulid.ID
	ID         pulid.ID
	Enabled    bool
}

type DeleteRequest struct {
	TenantInfo pagination.TenantInfo
	ActorID    pulid.ID
	ID         pulid.ID
}

func (s *Service) List(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*triage.Rule, error) {
	return s.rules.List(ctx, triageports.ListRulesRequest{TenantInfo: tenantInfo})
}

func (s *Service) Get(ctx context.Context, req triageports.GetRuleRequest) (*triage.Rule, error) {
	return s.rules.GetByID(ctx, req)
}

func (s *Service) Create(ctx context.Context, req SaveRequest) (*triage.Rule, error) {
	rule := sentenceOf(req.Rule)
	rule.OrganizationID = req.TenantInfo.OrgID
	rule.BusinessUnitID = req.TenantInfo.BuID
	rule.CreatedByID = req.ActorID
	rule.Enabled = req.Rule.Enabled

	if err := s.validate(ctx, req.TenantInfo, rule); err != nil {
		return nil, err
	}

	created, err := s.rules.Create(ctx, rule, triage.MaxRulesPerTenant)
	if err != nil {
		return nil, s.saveError(err)
	}

	s.rulesChanged(ctx, req.TenantInfo)
	s.logAudit(ctx, &auditEntry{
		tenantInfo: req.TenantInfo,
		actorID:    req.ActorID,
		operation:  permission.OpCreate,
		current:    created,
	})

	return created, nil
}

func (s *Service) Update(ctx context.Context, req *UpdateRequest) (*triage.Rule, error) {
	original, err := s.rules.GetByID(ctx, triageports.GetRuleRequest{
		TenantInfo: req.TenantInfo,
		ID:         req.ID,
	})
	if err != nil {
		return nil, err
	}

	rule := sentenceOf(req.Rule)
	rule.ID = original.ID
	rule.OrganizationID = original.OrganizationID
	rule.BusinessUnitID = original.BusinessUnitID
	rule.CreatedByID = original.CreatedByID
	rule.CreatedAt = original.CreatedAt
	rule.MatchCount = original.MatchCount
	rule.LastMatchedAt = original.LastMatchedAt
	rule.LastMessageID = original.LastMessageID
	rule.Enabled = req.Rule.Enabled
	rule.Version = req.Version

	if err = s.validate(ctx, req.TenantInfo, rule); err != nil {
		return nil, err
	}

	updated, err := s.rules.Update(ctx, rule)
	if err != nil {
		return nil, s.saveError(err)
	}

	s.rulesChanged(ctx, req.TenantInfo)
	s.logAudit(ctx, &auditEntry{
		tenantInfo: req.TenantInfo,
		actorID:    req.ActorID,
		operation:  permission.OpUpdate,
		current:    updated,
		previous:   original,
	})

	return updated, nil
}

func (s *Service) SetEnabled(ctx context.Context, req *SetEnabledRequest) (*triage.Rule, error) {
	original, err := s.rules.GetByID(ctx, triageports.GetRuleRequest{
		TenantInfo: req.TenantInfo,
		ID:         req.ID,
	})
	if err != nil {
		return nil, err
	}
	if original.Enabled == req.Enabled {
		return original, nil
	}
	if req.Enabled && original.Action == inboundmessage.NovaActionAssign {
		if err = s.validateAssignee(ctx, req.TenantInfo, original.AssigneeID); err != nil {
			return nil, err
		}
	}

	updated, err := s.rules.SetEnabled(ctx, triageports.SetRuleEnabledRequest{
		TenantInfo: req.TenantInfo,
		ID:         req.ID,
		Enabled:    req.Enabled,
	})
	if err != nil {
		return nil, err
	}

	s.rulesChanged(ctx, req.TenantInfo)
	s.logAudit(ctx, &auditEntry{
		tenantInfo: req.TenantInfo,
		actorID:    req.ActorID,
		operation:  permission.OpUpdate,
		current:    updated,
		previous:   original,
	})

	return updated, nil
}

func (s *Service) Delete(ctx context.Context, req *DeleteRequest) error {
	original, err := s.rules.GetByID(ctx, triageports.GetRuleRequest{
		TenantInfo: req.TenantInfo,
		ID:         req.ID,
	})
	if err != nil {
		return err
	}

	if err = s.rules.Delete(ctx, triageports.GetRuleRequest{
		TenantInfo: req.TenantInfo,
		ID:         req.ID,
	}); err != nil {
		return err
	}

	s.rulesChanged(ctx, req.TenantInfo)
	s.logAudit(ctx, &auditEntry{
		tenantInfo: req.TenantInfo,
		actorID:    req.ActorID,
		operation:  permission.OpDelete,
		previous:   original,
	})

	return nil
}

func (s *Service) Teach(
	ctx context.Context,
	req inboundmessageports.TeachRequest, //nolint:gocritic // the TriageRuleTeacher port takes its request by value
) (*triage.Rule, error) {
	if req.SenderDomain == "" {
		return nil, errortypes.NewValidationError(
			"sender", errortypes.ErrRequired, "Nova can only learn a rule for a known sender",
		)
	}

	rule := &triage.Rule{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		Classification: req.Classification,
		SenderScope:    triage.SenderDomain,
		Sender:         req.SenderDomain,
		MailboxID:      req.MailboxID,
		Action:         req.Action,
		Enabled:        true,
		CreatedByID:    req.ActorID,
	}
	if req.Action == inboundmessage.NovaActionAssign {
		rule.AssigneeID = req.ActorID
	}
	rule.Normalize()

	existing, err := s.rules.FindBySentence(ctx, triageports.FindRuleBySentenceRequest{
		TenantInfo: req.TenantInfo,
		Sentence:   rule,
	})
	if err != nil {
		return nil, err
	}

	taught := existing
	if existing == nil {
		if taught, err = s.Create(ctx, SaveRequest{
			TenantInfo: req.TenantInfo,
			ActorID:    req.ActorID,
			Rule:       rule,
		}); err != nil {
			return nil, err
		}
	} else if !existing.Enabled {
		if taught, err = s.SetEnabled(ctx, &SetEnabledRequest{
			TenantInfo: req.TenantInfo,
			ActorID:    req.ActorID,
			ID:         existing.ID,
			Enabled:    true,
		}); err != nil {
			return nil, err
		}
	}

	if req.MessageID.IsNotNil() {
		activity := &triage.Activity{
			OrganizationID: req.TenantInfo.OrgID,
			BusinessUnitID: req.TenantInfo.BuID,
			MessageID:      req.MessageID,
			Kind:           triage.ActivityRuleCreated,
			RecordID:       taught.ID,
		}
		activity.ByPerson(req.ActorID).About(string(taught.Action), taught.Sender)
		if err = s.activities.RecordActivity(ctx, activity); err != nil {
			s.l.Warn("could not record the rule a person taught",
				zap.String("ruleId", taught.ID.String()), zap.Error(err))
		}
	}

	return taught, nil
}

func sentenceOf(input *triage.Rule) *triage.Rule {
	rule := &triage.Rule{
		Classification: input.Classification,
		SenderScope:    input.SenderScope,
		Sender:         input.Sender,
		MailboxID:      input.MailboxID,
		Action:         input.Action,
		AssigneeID:     input.AssigneeID,
	}
	rule.Normalize()

	return rule
}

func (s *Service) validate(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	rule *triage.Rule,
) error {
	multiErr := errortypes.NewMultiError()
	rule.Validate(multiErr)
	if multiErr.HasErrors() {
		return multiErr
	}

	if rule.MailboxID.IsNotNil() {
		if _, err := s.mailboxes.GetByID(ctx, inboundmessageports.GetMailboxByIDRequest{
			ID:         rule.MailboxID,
			TenantInfo: tenantInfo,
		}); err != nil {
			if !errortypes.IsNotFoundError(err) && !dberror.IsNotFoundError(err) {
				return err
			}
			multiErr.Add("mailboxId", errortypes.ErrInvalid, "That source is not one of yours")
		}
	}

	if rule.Action == inboundmessage.NovaActionAssign {
		allowed, err := mayReadInbox(ctx, s.permissions, tenantInfo, rule.AssigneeID)
		if err != nil {
			return err
		}
		if !allowed {
			multiErr.Add("assigneeId", errortypes.ErrInvalid, assigneeCannotRead)
		}
	}

	existing, err := s.rules.FindBySentence(ctx, triageports.FindRuleBySentenceRequest{
		TenantInfo: tenantInfo,
		Sentence:   rule,
	})
	if err != nil {
		return err
	}
	if existing != nil {
		multiErr.Add("classification", errortypes.ErrDuplicate, duplicateSentence)
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func (s *Service) validateAssignee(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	assigneeID pulid.ID,
) error {
	allowed, err := mayReadInbox(ctx, s.permissions, tenantInfo, assigneeID)
	if err != nil {
		return err
	}
	if !allowed {
		multiErr := errortypes.NewMultiError()
		multiErr.Add("assigneeId", errortypes.ErrInvalid, assigneeCannotRead)

		return multiErr
	}

	return nil
}

func (s *Service) saveError(err error) error {
	switch {
	case errors.Is(err, triageports.ErrTooManyRules):
		return errortypes.NewBusinessError(
			"An organization can have at most {0} rules. Turn one off or delete one first.",
			triage.MaxRulesPerTenant,
		)
	case dberror.IsUniqueConstraintViolation(err):
		multiErr := errortypes.NewMultiError()
		multiErr.Add("classification", errortypes.ErrDuplicate, duplicateSentence)

		return multiErr
	default:
		return err
	}
}

type auditEntry struct {
	tenantInfo pagination.TenantInfo
	actorID    pulid.ID
	operation  permission.Operation
	current    *triage.Rule
	previous   *triage.Rule
}

func (s *Service) logAudit(_ context.Context, entry *auditEntry) {
	if s.audit == nil {
		return
	}

	params := &auditports.LogActionParams{
		Resource:       permission.ResourceInboundMessage,
		Operation:      entry.operation,
		UserID:         entry.actorID,
		OrganizationID: entry.tenantInfo.OrgID,
		BusinessUnitID: entry.tenantInfo.BuID,
	}
	if entry.current != nil {
		params.ResourceID = entry.current.ID.String()
		params.CurrentState = jsonutils.MustToJSON(entry.current)
	}
	if entry.previous != nil {
		params.ResourceID = entry.previous.ID.String()
		params.PreviousState = jsonutils.MustToJSON(entry.previous)
	}

	if err := s.audit.LogAction(params); err != nil {
		s.l.Warn("could not audit a triage rule change",
			zap.String("ruleId", params.ResourceID), zap.Error(err))
	}
}
