package agentturn

import (
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/shared/pulid"
)

type RuntimeAnchor struct {
	Kind  string       `json:"kind"`
	ID    string       `json:"id"`
	Label string       `json:"label,omitempty"`
	Facts []AnchorFact `json:"facts,omitempty"`
	Note  string       `json:"note,omitempty"`
}

type AnchorFact struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	SeenAt int64  `json:"seenAt,omitempty"`
	Stale  bool   `json:"stale,omitempty"`
}

type AttachmentReading string

const (
	AttachmentReadingDone    AttachmentReading = "done"
	AttachmentReadingPending AttachmentReading = "pending"
	AttachmentReadingFailed  AttachmentReading = "failed"
)

type RuntimeAttachment struct {
	DocumentID  string
	FileName    string
	ContentType string
	FileSize    int64
	PageCount   int
	Kind        string
	Reading     AttachmentReading
	Text        string
	PoorlyRead  bool
	// Shown is the link to the card beside the conversation that shows what
	// was read off the file, empty while there is none.
	Shown string
}

func (rc *RuntimeContext) MemoryRecords() []agent.EntityRef {
	records := make([]agent.EntityRef, 0, 2+len(rc.Mentions)+len(rc.DelegatorRecords))
	seen := make(map[agent.EntityRef]struct{}, cap(records))
	add := func(kind, id string) {
		kind, id = strings.TrimSpace(kind), strings.TrimSpace(id)
		if kind == "" || id == "" {
			return
		}
		ref := agent.EntityRef{Type: kind, ID: id}
		if _, ok := seen[ref]; ok {
			return
		}
		seen[ref] = struct{}{}
		records = append(records, ref)
	}

	if rc.Subject != nil {
		add(string(rc.Subject.Type), rc.Subject.ID)
	}
	if rc.Page != nil {
		add(rc.Page.EntityType, rc.Page.EntityID)
	}
	for _, mention := range rc.Mentions {
		add(mention.Type, mention.ID)
	}
	for _, record := range rc.DelegatorRecords {
		add(record.Type, record.ID)
	}

	return records
}

type RuntimeUser struct {
	Name  string
	Email string
	Roles []string
	// Timezone is the person's own clock, which a schedule they ask for runs
	// on. It is named to the agent only where it is not the organization's.
	Timezone string
}

type PageContext = agent.PageContext

type RuntimeSubject struct {
	Type            agent.SubjectType
	ID              string
	Label           string
	Notes           string
	OutsideAuthored agent.TaintSource
}

// RuntimeMention is a record the person pointed at by name while asking.
type RuntimeMention = agent.EntityRef

type ToolSummary struct {
	Name        string
	Description string
	Tier        agent.AutonomyTier
	Query       bool
	Lookup      bool
	// Loaded says the tool's schema is on this turn's request. Meaningful
	// only when the turn disclosed a subset; the rest load through find_tools.
	Loaded bool
	// PersonalRunsUnasked says a call that touches only the person's own
	// records runs at once while they are in the conversation, whatever
	// the static tier says: remember for a memory kept for them alone.
	PersonalRunsUnasked bool
	// Recipe is the tools a task with this one is done with, in order.
	Recipe []string
	// AlwaysProposes says every call of the tool records a proposal on this
	// agent: the tool's own ceiling, its egress or the agent's ceiling keeps
	// it from ever running on its own.
	AlwaysProposes bool
	Escalates      bool
}

func ToolNamesOf(summaries []ToolSummary) []string {
	names := make([]string, 0, len(summaries))
	for idx := range summaries {
		names = append(names, summaries[idx].Name)
	}

	return names
}

type RuntimeContext struct {
	OrganizationName string
	BusinessUnitName string
	Timezone         string
	Now              int64
	Trigger          agent.RunTrigger
	Surface          agent.Surface
	SurfaceGuide     *RuntimePage
	User             *RuntimeUser
	Subject          *RuntimeSubject
	Page             *PageContext
	// Attachments and Mentions are what the person handed over with the
	// message: files, and records named from the composer.
	Attachments []RuntimeAttachment
	Mentions    []RuntimeMention
	// Facts are what the person pinned for the agents to keep in mind for
	// the whole conversation. They are rendered into the system prompt, not
	// the history, so no trimming of the history can drop one.
	Facts []string
	Tools []ToolSummary
	// Memories is what the organization has recorded for its agents that this
	// turn may read, best first: the organization-wide ones, any about this
	// agent's tools, and any about the records the turn is about.
	// BuildSystemPrompt carries as many as the agent's memory budget holds.
	Memories []*agent.Memory
	// MemorySubjects are the records whose memories the turn reads, each
	// marked as the record the turn is about or one it names.
	MemorySubjects []agent.MemorySubject
	// MemoryRelevance says which of Memories bear on what the turn asked.
	// Facts and Corrections to tools not in play are carried only when they
	// do, and only those that do are counted and shown as used.
	MemoryRelevance agent.MemoryRelevance
	// MemoriesHeldBack counts the candidates the fit left out, so the prompt
	// can say agent_list_memories still finds them after Memories is narrowed to
	// what was carried.
	MemoriesHeldBack int
	// DelegatorRecords are the records the turn that handed this one its
	// task was about, so their memories reach the agent doing the task.
	DelegatorRecords []agent.EntityRef
	// ToolsDisclosed reports that the turn opened with a subset of the agent's
	// tools and can load the rest on demand. The prompt has to say so, because
	// the alternative is a model that reads a short tool list as the limit of
	// what the system can do and tells the person it is not possible.
	ToolsDisclosed bool
	ReachesBeyond  bool
	ToolMap        []ToolNamespace
	Frame          *TaskFrame
	// PendingProposals are the writes earlier turns proposed that the person
	// has not decided yet. The model is told so it points them at the card
	// rather than proposing the same change again.
	PendingProposals []PendingProposal
	DecisionRequests bool
	// Artifacts says the conversation keeps what the turn produces beside it:
	// lists as tables, records as cards, and documents the model publishes.
	Artifacts bool
	// Inline says the reply is read on its own with nothing beside it — a
	// quick question in the command palette — so everything it shows is
	// written into it.
	Inline bool
	// PageGuide is what the product guide says about the page in Page: its
	// name, where it sits and what it is for. A raw path tells the model
	// nothing about what the person is looking at.
	PageGuide *RuntimePage
	// Guide says a person is in the conversation and the agent can read the
	// product guide and move the app, so the prompt says how to use both.
	Guide bool
	// Delegates are the agents this one may hand a task to on this turn:
	// those on its allowlist the person may use. Empty for a run nobody is
	// watching and for a turn that is itself working for another agent.
	Delegates []RuntimeDelegate
	// DelegatedBy names the agent that handed this turn its task, when it is
	// working for another agent rather than for the person directly.
	DelegatedBy string
	LooksUpOnly bool
	Anchors     []RuntimeAnchor
	// OrganizationProfile is what the organization's records and its people's
	// guidance say about the business, written up each night. It is the same
	// on every turn of the day, so it sits in the cached part of the prompt.
	OrganizationProfile string
	// UserProfile is what the person's own records say they work on.
	UserProfile string
}

type ToolNamespace struct {
	Name        string
	Covers      []string
	MoreCovered int
	Tools       int
}

// RuntimeDelegate is an agent the running one may hand a task to, as its
// prompt and its delegate_task tool describe it.
type RuntimeDelegate struct {
	ID          pulid.ID `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Icon        string   `json:"icon,omitempty"`
	Accent      string   `json:"accent,omitempty"`
	// Tools names a few of the tools it holds, so the model can tell which
	// agent to ask for what.
	Tools []string `json:"tools,omitempty"`
}

// AsDelegate is the agent as another agent's turn names it, without the
// tools it holds, which depend on the person.
func AsDelegate(d *agentdefinition.Definition) RuntimeDelegate {
	return RuntimeDelegate{
		ID:          d.ID,
		Name:        d.Name,
		Description: d.Description,
		Icon:        d.ChosenIcon(),
		Accent:      d.ResolvedAccent(),
	}
}

// RuntimePage is one page of Trenova as the product guide describes it.
type RuntimePage struct {
	Name     string
	Location string
	Summary  string
}

// PendingProposal is one undecided proposal as the prompt names it.
type PendingProposal struct {
	ProposalID pulid.ID
	PlanID     pulid.ID
	ToolName   string
	Rationale  string
}

type TaskFrame struct {
	Goal     string   `json:"goal"`
	Need     string   `json:"need,omitempty"`
	Steps    []string `json:"steps,omitempty"`
	Done     string   `json:"done,omitempty"`
	Playbook string   `json:"playbook,omitempty"`
	Inferred bool     `json:"inferred,omitempty"`
}
