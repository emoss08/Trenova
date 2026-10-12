package triage

import (
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/shared/pulid"
)

type ActionKind string

const (
	ActionFile         = ActionKind("File")
	ActionBuildLoad    = ActionKind("BuildLoad")
	ActionReply        = ActionKind("Reply")
	ActionCheckStops   = ActionKind("CheckStops")
	ActionLogException = ActionKind("LogException")
	ActionBlockSender  = ActionKind("BlockSender")
	ActionLink         = ActionKind("Link")
)

type ReplyPurpose string

const (
	ReplyConfirmTender = ReplyPurpose("ConfirmTender")
	ReplyAcceptTender  = ReplyPurpose("AcceptTender")
	ReplyAnswerDispute = ReplyPurpose("AnswerDispute")
	ReplyReturnSigned  = ReplyPurpose("ReturnSigned")
	ReplyAskResend     = ReplyPurpose("AskResend")
	ReplySendEta       = ReplyPurpose("SendEta")
)

type DeckGroup string

const (
	DeckFile   = DeckGroup("File")
	DeckDraft  = DeckGroup("Draft")
	DeckDecide = DeckGroup("Decide")
	DeckLink   = DeckGroup("Link")
	DeckHeld   = DeckGroup("Held")
)

func AllDeckGroups() []DeckGroup {
	return []DeckGroup{DeckFile, DeckDraft, DeckDecide, DeckLink, DeckHeld}
}

func (g DeckGroup) EstimateSeconds() int {
	switch g {
	case DeckFile:
		return 60
	case DeckDraft, DeckLink:
		return 120
	case DeckDecide:
		return 240
	case DeckHeld:
		return 30
	default:
		return 120
	}
}

type Action struct {
	ID              string
	Kind            ActionKind
	Done            bool
	NovaCanDo       bool
	AttachmentID    pulid.ID
	FileName        string
	DocumentKind    inboundmessage.AttachmentKind
	FiledShipmentID pulid.ID
	FiledAs         string
	ShipmentID      pulid.ID
	ReplyPurpose    ReplyPurpose
	RefusedFileName string
	ReplyID         pulid.ID
	ReplyStatus     ReplyStatus
	ReplySentAt     int64
	LoadDrafted     bool
	CreatedShipment pulid.ID
	StopsDecision   inboundmessage.StopsDecision
	ExceptionKind   inboundmessage.ExceptionKind
	CompletedAt     int64
}

type ActionInput struct {
	Message *inboundmessage.InboundMessage
	Reply   *Reply
}

var paperwork = map[inboundmessage.Classification]struct{}{
	inboundmessage.ClassificationProofOfDelivery:  {},
	inboundmessage.ClassificationRateConfirmation: {},
	inboundmessage.ClassificationInvoice:          {},
}

func NeedsRecord(class inboundmessage.Classification) bool {
	switch class {
	case inboundmessage.ClassificationProofOfDelivery,
		inboundmessage.ClassificationRateConfirmation,
		inboundmessage.ClassificationInvoice,
		inboundmessage.ClassificationDetentionDispute,
		inboundmessage.ClassificationStatusRequest:
		return true
	default:
		return false
	}
}

func FilesAttachments(class inboundmessage.Classification) bool {
	switch class {
	case inboundmessage.ClassificationDetentionDispute, inboundmessage.ClassificationStatusRequest:
		return false
	default:
		return true
	}
}

func Actions(in ActionInput) []Action {
	m := in.Message
	if !m.NovaRead {
		return nil
	}

	state := m.ActionState
	actions := make([]Action, 0, 4)

	if m.Status == inboundmessage.StatusQuarantined {
		return append(actions, Action{
			ID:          "block",
			Kind:        ActionBlockSender,
			Done:        state.SenderBlockedAt > 0,
			CompletedAt: state.SenderBlockedAt,
		})
	}

	link := Action{
		ID:          "link",
		Kind:        ActionLink,
		Done:        state.LinkedAt > 0,
		CompletedAt: state.LinkedAt,
	}
	if m.FailureText != "" && !m.HasRecord() {
		return append(actions, link)
	}

	class := m.Classification
	if NeedsRecord(class) && !m.HasRecord() && state.LinkedAt == 0 {
		actions = append(actions, link)
		if class == inboundmessage.ClassificationStatusRequest {
			return actions
		}
	}

	if FilesAttachments(class) {
		for _, attachment := range m.Attachments {
			if !attachment.Readable() {
				continue
			}
			actions = append(actions, fileAction(m, attachment))
		}
	}

	switch class {
	case inboundmessage.ClassificationTender:
		actions = append(actions, loadAction(m))
		purpose := ReplyConfirmTender
		if hasReadableAttachment(m) {
			purpose = ReplyAcceptTender
		}
		actions = append(actions, replyAction(in.Reply, purpose, false))
	case inboundmessage.ClassificationDetentionDispute:
		if m.MatchedShipmentID.IsNotNil() {
			actions = append(actions, Action{
				ID:            "stops",
				Kind:          ActionCheckStops,
				Done:          state.StopsDecision != "",
				ShipmentID:    m.MatchedShipmentID,
				StopsDecision: state.StopsDecision,
				CompletedAt:   state.StopsDecidedAt,
			})
		}
		actions = append(actions, replyAction(in.Reply, ReplyAnswerDispute, false))
	case inboundmessage.ClassificationRateConfirmation:
		actions = append(actions, replyAction(in.Reply, ReplyReturnSigned, false))
	case inboundmessage.ClassificationProofOfDelivery:
		if hasWarningFact(m) && m.MatchedShipmentID.IsNotNil() {
			actions = append(actions, Action{
				ID:            "exception",
				Kind:          ActionLogException,
				Done:          state.ExceptionLoggedAt > 0,
				ShipmentID:    m.MatchedShipmentID,
				ExceptionKind: state.ExceptionKind,
				CompletedAt:   state.ExceptionLoggedAt,
			})
		}
		if refused := firstRefused(m); refused != "" {
			reply := replyAction(in.Reply, ReplyAskResend, false)
			reply.RefusedFileName = refused
			actions = append(actions, reply)
		}
	case inboundmessage.ClassificationStatusRequest:
		actions = append(actions, replyAction(in.Reply, ReplySendEta, true))
	case inboundmessage.ClassificationInvoice, inboundmessage.ClassificationOther:
	}

	for i := range actions {
		if actions[i].ShipmentID.IsNil() && actions[i].Kind != ActionLink {
			actions[i].ShipmentID = m.MatchedShipmentID
		}
	}

	return actions
}

func NextOpen(actions []Action) (Action, bool) {
	for _, action := range actions {
		if !action.Done {
			return action, true
		}
	}

	return Action{}, false
}

func GroupFor(m *inboundmessage.InboundMessage, actions []Action) (DeckGroup, Action, bool) {
	if m.Status == inboundmessage.StatusQuarantined {
		next, _ := NextOpen(actions)

		return DeckHeld, next, true
	}
	next, ok := NextOpen(actions)
	if !ok {
		return "", Action{}, false
	}

	switch {
	case next.Kind == ActionFile && m.MatchedShipmentID.IsNotNil():
		return DeckFile, next, true
	case next.NovaCanDo:
		return DeckDraft, next, true
	case next.Kind == ActionLink:
		return DeckLink, next, true
	default:
		return DeckDecide, next, true
	}
}

func fileAction(m *inboundmessage.InboundMessage, a *inboundmessage.InboundAttachment) Action {
	return Action{
		ID:              "file:" + a.ID.String(),
		Kind:            ActionFile,
		Done:            a.Filed(),
		AttachmentID:    a.ID,
		FileName:        a.FileName,
		DocumentKind:    a.Kind,
		FiledShipmentID: a.FiledShipmentID,
		FiledAs:         a.FiledAs,
		ShipmentID:      m.MatchedShipmentID,
		CompletedAt:     a.FiledAt,
	}
}

func loadAction(m *inboundmessage.InboundMessage) Action {
	state := m.ActionState

	return Action{
		ID:              "load",
		Kind:            ActionBuildLoad,
		Done:            state.CreatedShipmentID.IsNotNil(),
		NovaCanDo:       m.LoadDraft == nil && state.CreatedShipmentID.IsNil(),
		LoadDrafted:     m.LoadDraft != nil,
		CreatedShipment: state.CreatedShipmentID,
		CompletedAt:     state.CreatedShipmentAt,
	}
}

func replyAction(reply *Reply, purpose ReplyPurpose, novaDrafts bool) Action {
	action := Action{
		ID:           "reply",
		Kind:         ActionReply,
		ReplyPurpose: purpose,
		NovaCanDo:    novaDrafts && reply == nil,
	}
	if reply != nil {
		action.ReplyID = reply.ID
		action.ReplyStatus = reply.Status
		action.Done = reply.Sent()
		action.ReplySentAt = reply.SentAt
		action.CompletedAt = reply.SentAt
	}

	return action
}

func hasReadableAttachment(m *inboundmessage.InboundMessage) bool {
	for _, attachment := range m.Attachments {
		if attachment.Readable() {
			return true
		}
	}

	return false
}

func firstRefused(m *inboundmessage.InboundMessage) string {
	for _, attachment := range m.Attachments {
		if attachment.FailureText != "" {
			return attachment.FileName
		}
	}

	return ""
}

func hasWarningFact(m *inboundmessage.InboundMessage) bool {
	for _, fact := range m.Facts {
		if fact.Tone == inboundmessage.FactToneWarning && !fact.Flagged {
			return true
		}
	}

	return false
}

func IsPaperwork(class inboundmessage.Classification) bool {
	_, ok := paperwork[class]

	return ok
}
