package conversation

// Role identifies who produced a message.
type Role string

const (
	RoleUser      = Role("User")
	RoleAssistant = Role("Assistant")
	// RoleTool carries a tool's result. It is persisted so a conversation can be
	// replayed and audited, not just re-read.
	RoleTool = Role("Tool")
)

func (r Role) IsValid() bool {
	switch r {
	case RoleUser, RoleAssistant, RoleTool:
		return true
	default:
		return false
	}
}

// MessageKind separates what a person wrote from what the application wrote on
// their behalf. A decision note is the input of the turn that follows a
// decision on a proposal: it is sent to the model like any user message, and
// shown to the person as a note rather than as words they typed.
type MessageKind string

const (
	MessageKindMessage      = MessageKind("Message")
	MessageKindDecisionNote = MessageKind("DecisionNote")
	// MessageKindDelegated is a step another agent took on a task the
	// conversation's agent handed it: the task, its model's messages and its
	// tool results. The conversation shows them nested under the delegate
	// call; the model is never sent them again, because the agent that
	// delegated only ever saw its own call and the answer it got back.
	MessageKindDelegated = MessageKind("Delegated")
	// MessageKindSchedule is a request the person asked to have repeated. It
	// is kept in the person's words with the schedule it made, and drawn as
	// that schedule's card. The model is never sent it: nothing answered it,
	// and each run asks the request afresh in a turn of its own.
	MessageKindSchedule = MessageKind("Schedule")
	// MessageKindCompaction is the summary a compaction wrote of the
	// conversation before it. The model reads it in place of every message
	// it stands in for, and the thread shows it as the line where the
	// conversation was compacted.
	MessageKindCompaction = MessageKind("Compaction")
	// MessageKindHandoff marks, in the conversation a person handed off, that
	// they took it to another agent and what went with it. The conversation
	// shows it as a card; the model is never sent it.
	MessageKindHandoff = MessageKind("Handoff")
	// MessageKindHandoffBrief opens the conversation a hand-off started: the
	// summary, pinned facts and pinned artifacts carried over. It is sent to
	// the model like a person's message, and shown as a note.
	MessageKindHandoffBrief = MessageKind("HandoffBrief")
)

func (k MessageKind) IsValid() bool {
	switch k {
	case MessageKindMessage, MessageKindDecisionNote, MessageKindDelegated,
		MessageKindSchedule, MessageKindCompaction, MessageKindHandoff, MessageKindHandoffBrief:
		return true
	default:
		return false
	}
}

// AllMessageKinds is every kind a message may be, in the order they were
// added.
func AllMessageKinds() []MessageKind {
	return []MessageKind{
		MessageKindMessage,
		MessageKindDecisionNote,
		MessageKindDelegated,
		MessageKindSchedule,
		MessageKindCompaction,
		MessageKindHandoff,
		MessageKindHandoffBrief,
	}
}

// ModelHiddenKinds are the kinds a conversation keeps and never replays to
// the model.
func ModelHiddenKinds() []MessageKind {
	return []MessageKind{MessageKindDelegated, MessageKindSchedule, MessageKindHandoff}
}

// ThreadStatus is a conversation's lifecycle.
type ThreadStatus string

const (
	ThreadStatusActive   = ThreadStatus("Active")
	ThreadStatusArchived = ThreadStatus("Archived")
)

func (s ThreadStatus) IsValid() bool {
	switch s {
	case ThreadStatusActive, ThreadStatusArchived:
		return true
	default:
		return false
	}
}

// ThreadOrigin is where a conversation was started. The Desk shows a
// person's own conversations; a quick question asked from the command
// palette stays out of that list until the person keeps it, and one opened
// from the watchtower or a briefing says so, since it began about something.
type ThreadOrigin string

const (
	ThreadOriginPanel      = ThreadOrigin("Panel")
	ThreadOriginDesk       = ThreadOrigin("Desk")
	ThreadOriginAsk        = ThreadOrigin("Ask")
	ThreadOriginWatchtower = ThreadOrigin("Watchtower")
	ThreadOriginBriefing   = ThreadOrigin("Briefing")
	ThreadOriginImport     = ThreadOrigin("Import")
	ThreadOriginFormula    = ThreadOrigin("Formula")
)

func AllThreadOrigins() []ThreadOrigin {
	return []ThreadOrigin{
		ThreadOriginPanel,
		ThreadOriginDesk,
		ThreadOriginAsk,
		ThreadOriginWatchtower,
		ThreadOriginBriefing,
		ThreadOriginImport,
		ThreadOriginFormula,
	}
}

func (o ThreadOrigin) IsValid() bool {
	switch o {
	case ThreadOriginPanel, ThreadOriginDesk, ThreadOriginAsk,
		ThreadOriginWatchtower, ThreadOriginBriefing,
		ThreadOriginImport, ThreadOriginFormula:
		return true
	default:
		return false
	}
}

// Listed reports whether the Desk's thread rail shows a conversation of
// this origin. A quick question is not listed until it is kept.
func (o ThreadOrigin) Listed() bool {
	return o != ThreadOriginAsk && !o.PageBound()
}

func (o ThreadOrigin) Keepable() bool {
	return o == ThreadOriginAsk
}

func (o ThreadOrigin) PageBound() bool {
	return o == ThreadOriginImport || o == ThreadOriginFormula
}

func PageBoundOrigins() []ThreadOrigin {
	return []ThreadOrigin{ThreadOriginImport, ThreadOriginFormula}
}

func UnlistedOrigins() []ThreadOrigin {
	all := AllThreadOrigins()
	out := make([]ThreadOrigin, 0, len(all))
	for _, origin := range all {
		if !origin.Listed() {
			out = append(out, origin)
		}
	}

	return out
}
