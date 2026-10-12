package triageruleservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/triageports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	filedNote   = "Nova filed its documents on the shipment, as a rule says."
	ignoredNote = "Nova ignored it, as a rule says."
	linkedNote  = "Linked by a rule to the customer or carrier the sender belongs to."
)

type EvaluatorParams struct {
	fx.In

	Logger      *zap.Logger
	DB          kernelports.DBConnection
	Rules       triageports.TriageRuleRepository
	Store       triageports.TriageMessageRepository
	Messages    inboundmessageports.InboundMessageRepository
	Parties     inboundmessageports.InboundPartyFinder
	Permissions permissionports.PermissionEngine
	Cache       triageports.TriageCache                `optional:"true"`
	Filer       inboundmessageports.TriageFiler        `optional:"true"`
	Drafter     inboundmessageports.TriageDrafter      `optional:"true"`
	Replies     inboundmessageports.InboundReplyReader `optional:"true"`
	Conflicts   triageports.TriageConflictChecker      `optional:"true"`
}

type Evaluator struct {
	l           *zap.Logger
	db          kernelports.DBConnection
	rules       triageports.TriageRuleRepository
	store       triageports.TriageMessageRepository
	messages    inboundmessageports.InboundMessageRepository
	parties     inboundmessageports.InboundPartyFinder
	permissions permissionports.PermissionEngine
	cache       triageports.TriageCache
	filer       inboundmessageports.TriageFiler
	drafter     inboundmessageports.TriageDrafter
	replies     inboundmessageports.InboundReplyReader
	conflicts   triageports.TriageConflictChecker
	now         func() int64
}

func NewEvaluator(p EvaluatorParams) *Evaluator {
	return &Evaluator{
		l:           p.Logger.Named("service.triage-rule-evaluator"),
		db:          p.DB,
		rules:       p.Rules,
		store:       p.Store,
		messages:    p.Messages,
		parties:     p.Parties,
		permissions: p.Permissions,
		cache:       p.Cache,
		filer:       p.Filer,
		drafter:     p.Drafter,
		replies:     p.Replies,
		conflicts:   p.Conflicts,
		now:         timeutils.NowUnix,
	}
}

var _ inboundmessageports.TriageRuleEvaluator = (*Evaluator)(nil)

type outcome struct {
	settle      inboundmessage.Status
	note        string
	ownerID     pulid.ID
	customerID  pulid.ID
	carrierID   pulid.ID
	matchReason string
}

func Eligible(message *inboundmessage.InboundMessage) bool {
	return message.NovaRead &&
		message.Status == inboundmessage.StatusInReview &&
		message.Parked == "" &&
		message.FailureText == "" &&
		message.Classification != ""
}

func tenantOf(message *inboundmessage.InboundMessage) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: message.OrganizationID, BuID: message.BusinessUnitID}
}

func (e *Evaluator) ApplyRules(
	ctx context.Context,
	message *inboundmessage.InboundMessage,
) (*inboundmessageports.RuleApplication, error) {
	if !Eligible(message) {
		return nil, nil //nolint:nilnil // nothing applies to a message no rule may act on
	}
	if message.HandledByRuleID.IsNotNil() {
		return e.replay(ctx, message)
	}

	tenantInfo := tenantOf(message)
	rules, err := enabledRules(ctx, ruleSource{l: e.l, rules: e.rules, cache: e.cache}, tenantInfo)
	if err != nil {
		return nil, err
	}

	rule := triage.FirstMatch(rules, message)
	if rule == nil {
		return nil, nil //nolint:nilnil // no rule matched, which is not a failure
	}

	result, acted, err := e.act(ctx, rule, message)
	if err != nil {
		return nil, fmt.Errorf("apply rule %s: %w", rule.ID, err)
	}
	if !acted {
		return nil, nil //nolint:nilnil // the rule could not act, so the message waits on a person
	}

	claimed, err := e.claim(ctx, rule, message)
	if err != nil {
		return nil, err
	}
	if !claimed {
		return e.replay(ctx, message)
	}

	settle(message, rule, result, e.now())

	return &inboundmessageports.RuleApplication{
		Rule:    rule,
		Settled: result.settle != "",
	}, nil
}

func (e *Evaluator) replay(
	ctx context.Context,
	message *inboundmessage.InboundMessage,
) (*inboundmessageports.RuleApplication, error) {
	tenantInfo := tenantOf(message)
	fresh, err := e.messages.GetByID(ctx, inboundmessageports.GetInboundMessageByIDRequest{
		ID:         message.ID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}
	refresh(message, fresh)
	if message.HandledByRuleID.IsNil() {
		return nil, nil //nolint:nilnil // nothing was claimed after all
	}

	rule, err := e.rules.GetByID(ctx, triageports.GetRuleRequest{
		TenantInfo: tenantInfo,
		ID:         message.HandledByRuleID,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, nil //nolint:nilnil // the rule was deleted after it acted
		}

		return nil, err
	}

	result := e.replayed(ctx, rule, message)
	settle(message, rule, result, e.now())

	return &inboundmessageports.RuleApplication{Rule: rule, Settled: result.settle != ""}, nil
}

func (e *Evaluator) replayed(
	ctx context.Context,
	rule *triage.Rule,
	message *inboundmessage.InboundMessage,
) outcome {
	switch rule.Action {
	case inboundmessage.NovaActionFileOnShipment, inboundmessage.NovaActionFileAndQueuePay:
		return outcome{settle: inboundmessage.StatusActioned, note: filedNote}
	case inboundmessage.NovaActionIgnore:
		return outcome{settle: inboundmessage.StatusIgnored, note: ignoredNote}
	case inboundmessage.NovaActionAssign:
		return outcome{ownerID: rule.AssigneeID}
	case inboundmessage.NovaActionLinkBySender:
		result, _, err := e.link(ctx, message)
		if err != nil {
			e.l.Warn("could not find the sender's party again",
				zap.String("messageId", message.ID.String()), zap.Error(err))
		}

		return result
	case inboundmessage.NovaActionDraftReply, inboundmessage.NovaActionDraftLoad:
		return outcome{}
	default:
		return outcome{}
	}
}

func (e *Evaluator) act(
	ctx context.Context,
	rule *triage.Rule,
	message *inboundmessage.InboundMessage,
) (outcome, bool, error) {
	switch rule.Action {
	case inboundmessage.NovaActionFileOnShipment, inboundmessage.NovaActionFileAndQueuePay:
		return e.file(ctx, rule, message)
	case inboundmessage.NovaActionDraftReply:
		return e.draftReply(ctx, message)
	case inboundmessage.NovaActionDraftLoad:
		return e.draftLoad(ctx, message)
	case inboundmessage.NovaActionLinkBySender:
		return e.link(ctx, message)
	case inboundmessage.NovaActionIgnore:
		return outcome{settle: inboundmessage.StatusIgnored, note: ignoredNote}, true, nil
	case inboundmessage.NovaActionAssign:
		allowed, err := mayReadInbox(ctx, e.permissions, tenantOf(message), rule.AssigneeID)
		if err != nil || !allowed {
			return outcome{}, false, err
		}

		return outcome{ownerID: rule.AssigneeID}, true, nil
	default:
		return outcome{}, false, nil
	}
}

func (e *Evaluator) file(
	ctx context.Context,
	rule *triage.Rule,
	message *inboundmessage.InboundMessage,
) (outcome, bool, error) {
	if e.filer == nil || message.MatchedShipmentID.IsNil() ||
		!triage.FilesAttachments(message.Classification) {
		return outcome{}, false, nil
	}

	if e.conflicts != nil {
		conflicts, err := e.conflicts.CheckConflicts(ctx, message)
		if err != nil {
			return outcome{}, false, err
		}
		if blocksFiling(conflicts) {
			return outcome{}, false, nil
		}
	}

	filed, err := e.filer.FileOnMatchedShipment(ctx, inboundmessageports.FileOnMatchedRequest{
		Message:     message,
		ByNova:      true,
		QueueForPay: rule.Action == inboundmessage.NovaActionFileAndQueuePay,
	})
	if err != nil {
		return outcome{}, false, err
	}

	attachments, err := e.messages.ListAttachments(ctx, message.ID, tenantOf(message))
	if err != nil {
		return outcome{}, false, err
	}
	message.Attachments = attachments
	if filed == 0 && !AllReadableFiled(attachments) {
		return outcome{}, false, nil
	}

	return outcome{settle: inboundmessage.StatusActioned, note: filedNote}, true, nil
}

func blocksFiling(conflicts []inboundmessage.Conflict) bool {
	for _, conflict := range conflicts {
		if conflict.Kind == inboundmessage.ConflictDuplicateDocument && conflict.Resolution == "" {
			return true
		}
	}

	return false
}

func AllReadableFiled(attachments []*inboundmessage.InboundAttachment) bool {
	readable := 0
	for _, attachment := range attachments {
		if !attachment.Readable() {
			continue
		}
		readable++
		if !attachment.Filed() {
			return false
		}
	}

	return readable > 0
}

func (e *Evaluator) draftReply(
	ctx context.Context,
	message *inboundmessage.InboundMessage,
) (outcome, bool, error) {
	if e.drafter == nil {
		return outcome{}, false, nil
	}

	if e.replies != nil {
		latest, err := e.replies.LatestByMessageIDs(ctx, tenantOf(message), []pulid.ID{message.ID})
		if err != nil {
			return outcome{}, false, err
		}
		if latest[message.ID] != nil {
			return outcome{}, true, nil
		}
	}

	if err := e.drafter.DraftReplyFor(ctx, message); err != nil {
		return outcome{}, false, err
	}

	return outcome{}, true, e.refresh(ctx, message)
}

func (e *Evaluator) draftLoad(
	ctx context.Context,
	message *inboundmessage.InboundMessage,
) (outcome, bool, error) {
	if e.drafter == nil || message.ActionState.CreatedShipmentID.IsNotNil() {
		return outcome{}, false, nil
	}
	if message.LoadDraft != nil {
		return outcome{}, true, nil
	}

	if err := e.drafter.DraftLoadFor(ctx, message); err != nil {
		return outcome{}, false, err
	}

	return outcome{}, true, e.refresh(ctx, message)
}

func (e *Evaluator) link(
	ctx context.Context,
	message *inboundmessage.InboundMessage,
) (outcome, bool, error) {
	if message.MatchedCustomerID.IsNotNil() || message.MatchedCarrierID.IsNotNil() {
		return outcome{}, true, nil
	}

	tenantInfo := tenantOf(message)
	customerID, found, err := e.parties.FindCustomerByEmail(ctx, tenantInfo, message.FromAddress)
	if err != nil {
		return outcome{}, false, err
	}
	if found {
		return outcome{customerID: customerID, matchReason: linkedNote}, true, nil
	}

	carrierID, found, err := e.parties.FindCarrierByEmail(ctx, tenantInfo, message.FromAddress)
	if err != nil {
		return outcome{}, false, err
	}
	if found {
		return outcome{carrierID: carrierID, matchReason: linkedNote}, true, nil
	}

	return outcome{}, false, nil
}

func (e *Evaluator) refresh(ctx context.Context, message *inboundmessage.InboundMessage) error {
	fresh, err := e.messages.GetByID(ctx, inboundmessageports.GetInboundMessageByIDRequest{
		ID:         message.ID,
		TenantInfo: tenantOf(message),
	})
	if err != nil {
		return err
	}
	refresh(message, fresh)

	return nil
}

func refresh(message, fresh *inboundmessage.InboundMessage) {
	message.Version = fresh.Version
	message.LoadDraft = fresh.LoadDraft
	message.Conflicts = fresh.Conflicts
	message.ActionState = fresh.ActionState
	message.HandledByRuleID = fresh.HandledByRuleID
	message.HandledAction = fresh.HandledAction
}

func (e *Evaluator) claim(
	ctx context.Context,
	rule *triage.Rule,
	message *inboundmessage.InboundMessage,
) (bool, error) {
	tenantInfo := tenantOf(message)
	at := e.now()

	var claimed *triageports.ClaimedRule
	err := e.db.WithTx(ctx, kernelports.TxOptions{}, func(ctx context.Context, _ bun.Tx) error {
		var txErr error
		claimed, txErr = e.store.ClaimRule(ctx, triageports.ClaimRuleRequest{
			TenantInfo: tenantInfo,
			MessageID:  message.ID,
			RuleID:     rule.ID,
			Action:     rule.Action,
			At:         at,
		})
		if txErr != nil || !claimed.Claimed {
			return txErr
		}

		return e.rules.RecordMatch(ctx, triageports.RecordRuleMatchRequest{
			TenantInfo: tenantInfo,
			RuleID:     rule.ID,
			MessageID:  message.ID,
			At:         at,
		})
	})
	if err != nil {
		return false, err
	}
	if !claimed.Claimed {
		return false, nil
	}

	message.Version = claimed.Version
	rule.MatchCount++
	rule.LastMatchedAt = at
	rule.LastMessageID = message.ID

	return true, nil
}

func settle(
	message *inboundmessage.InboundMessage,
	rule *triage.Rule,
	result outcome,
	now int64,
) {
	message.HandledByRuleID = rule.ID
	message.HandledAction = rule.Action
	if result.settle != "" {
		message.Status = result.settle
		message.ReviewedAt = now
		message.ReviewedBy = pulid.Nil
		message.ReviewNote = result.note
	}
	if result.ownerID.IsNotNil() && message.OwnerID.IsNil() {
		message.OwnerID = result.ownerID
	}
	if result.customerID.IsNotNil() && message.MatchedCustomerID.IsNil() {
		message.MatchedCustomerID = result.customerID
		message.MatchReason = result.matchReason
	}
	if result.carrierID.IsNotNil() && message.MatchedCarrierID.IsNil() {
		message.MatchedCarrierID = result.carrierID
		message.MatchReason = result.matchReason
	}
}
