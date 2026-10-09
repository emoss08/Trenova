package agentdefinition

import (
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/shared/llmtokens"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

type memoryTier int

const (
	memoryTierDirectSubject memoryTier = iota
	memoryTierRelatedSubject
	memoryTierOrganizationInstruction
	memoryTierLoadedTool
	memoryTierRest
)

const (
	organizationMemoryHeading = "For the whole organization"
	personalMemoryHeading     = "For the person you are talking to"
	roleMemoryHeading         = "For everyone in their role"
	agentMemoryHeading        = "For this agent"
)

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

// MemoryFit is what a turn does with the memories it read. Carried is what
// the prompt holds. Used is what bore on the turn: memories about the records
// it is about, about a tool picked for it, or that share meaning or words
// with what it asked. ByTool holds the carried memories about a tool that
// was only on hand, which become used when the turn calls it. HeldBack
// counts the candidates left out, each still a recall_memory call away.
type MemoryFit struct {
	Carried  []*agent.Memory
	Used     []pulid.ID
	ByTool   map[string][]pulid.ID
	HeldBack int
}

// FitMemories picks what the prompt carries.
func (d *Definition) FitMemories(rc *RuntimeContext) []*agent.Memory {
	return d.PlanMemories(rc).Carried
}

// PlanMemories fits the memories that have not gone stale to the agent's
// token budget and MaxPromptMemories, best first, and says which of them
// bear on the turn. A memory in no tier of its own (a Fact, or a Correction
// to a tool not in play) is carried only when it bears on what was asked;
// when there was nothing to judge by, every candidate bears, as before.
func (d *Definition) PlanMemories(rc *RuntimeContext) MemoryFit {
	if rc == nil || len(rc.Memories) == 0 {
		return MemoryFit{HeldBack: heldBack(rc, 0)}
	}

	ranked := rankMemoriesForPrompt(rc, agent.WithoutStaleMemories(rc.Memories))
	budget := d.EffectiveMemoryTokenBudget()
	spent := 0
	headed := make(map[string]struct{}, len(ranked))
	fit := MemoryFit{
		Carried: make([]*agent.Memory, 0, min(len(ranked), MaxPromptMemories)),
		Used:    make([]pulid.ID, 0, min(len(ranked), MaxPromptMemories)),
	}
	for _, entry := range ranked {
		if len(fit.Carried) == MaxPromptMemories {
			break
		}
		if entry.tier == memoryTierRest && !entry.relevant {
			continue
		}

		memory := entry.memory
		var cost int
		if memory.DrawnFromOutside() {
			cost = llmtokens.Estimate(outsideMemoryLine(memory))
		} else {
			cost = llmtokens.Estimate(memoryLine(memory))
			if _, hasHeading := headed[memoryGroupKey(memory)]; !hasHeading {
				cost += llmtokens.Estimate(memoryHeading(memory))
			}
		}
		if spent+cost > budget {
			continue
		}
		spent += cost
		if !memory.DrawnFromOutside() {
			headed[memoryGroupKey(memory)] = struct{}{}
		}
		fit.Carried = append(fit.Carried, memory)
		fit.note(rc, entry)
	}
	fit.HeldBack = heldBack(rc, len(ranked)-len(fit.Carried))

	return fit
}

func heldBack(rc *RuntimeContext, left int) int {
	if rc == nil {
		return max(left, 0)
	}

	return max(left, 0) + rc.MemoriesHeldBack
}

func (f *MemoryFit) note(rc *RuntimeContext, entry rankedMemory) {
	memory := entry.memory
	if memory.ID.IsNil() {
		return
	}

	switch {
	case entry.tier == memoryTierDirectSubject,
		entry.tier == memoryTierRelatedSubject,
		entry.relevant:
		f.Used = append(f.Used, memory.ID)
	case entry.tier == memoryTierLoadedTool && rc.ToolsDisclosed:
		f.Used = append(f.Used, memory.ID)
	case entry.tier == memoryTierLoadedTool:
		if f.ByTool == nil {
			f.ByTool = make(map[string][]pulid.ID, 4)
		}
		tool := strings.TrimSpace(memory.ToolName)
		f.ByTool[tool] = append(f.ByTool[tool], memory.ID)
	}
}

type rankedMemory struct {
	memory   *agent.Memory
	tier     memoryTier
	rank     int
	relevant bool
}

func rankMemoriesForPrompt(rc *RuntimeContext, memories []*agent.Memory) []rankedMemory {
	relations := memoryRelations(rc.MemorySubjects)
	loaded := loadedToolNames(rc)

	ranked := make([]rankedMemory, 0, len(memories))
	for idx, memory := range memories {
		if memory == nil || strings.TrimSpace(memory.Content) == "" {
			continue
		}
		ranked = append(ranked, rankedMemory{
			memory:   memory,
			tier:     tierOf(memory, relations, loaded),
			rank:     idx,
			relevant: rc.MemoryRelevance.Bears(memory.ID),
		})
	}

	slices.SortStableFunc(ranked, func(a, b rankedMemory) int {
		if a.tier != b.tier {
			return int(a.tier) - int(b.tier)
		}
		if a.relevant != b.relevant {
			if a.relevant {
				return -1
			}
			return 1
		}
		if a.tier != memoryTierRest {
			if byKind := a.memory.Kind.Rank() - b.memory.Kind.Rank(); byKind != 0 {
				return byKind
			}
		}

		return a.rank - b.rank
	})

	return ranked
}

type subjectKey struct {
	kind agent.MemorySubjectType
	id   pulid.ID
}

func memoryRelations(subjects []agent.MemorySubject) map[subjectKey]agent.MemoryRelation {
	relations := make(map[subjectKey]agent.MemoryRelation, len(subjects))
	for _, subject := range subjects {
		key := subjectKey{kind: subject.Type, id: subject.ID}
		if existing, ok := relations[key]; ok && existing == agent.MemoryRelationDirect {
			continue
		}
		relations[key] = subject.Relation
	}

	return relations
}

func loadedToolNames(rc *RuntimeContext) map[string]struct{} {
	loaded := make(map[string]struct{}, len(rc.Tools))
	for _, tool := range rc.Tools {
		if !rc.ToolsDisclosed || tool.Loaded {
			loaded[tool.Name] = struct{}{}
		}
	}

	return loaded
}

func tierOf(
	memory *agent.Memory,
	relations map[subjectKey]agent.MemoryRelation,
	loaded map[string]struct{},
) memoryTier {
	if memory.SubjectType != "" && memory.SubjectID != nil {
		switch relations[subjectKey{kind: memory.SubjectType, id: *memory.SubjectID}] {
		case agent.MemoryRelationDirect:
			return memoryTierDirectSubject
		case agent.MemoryRelationRelated:
			return memoryTierRelatedSubject
		default:
			return memoryTierRest
		}
	}
	if tool := strings.TrimSpace(memory.ToolName); tool != "" {
		if _, ok := loaded[tool]; ok {
			return memoryTierLoadedTool
		}

		return memoryTierRest
	}
	if memory.Kind.Followed() {
		return memoryTierOrganizationInstruction
	}

	return memoryTierRest
}

func memoryGroupKey(memory *agent.Memory) string {
	switch {
	case memory.SubjectType != "" && memory.SubjectID != nil:
		return string(memory.SubjectType) + ":" + memory.SubjectID.String()
	case strings.TrimSpace(memory.ToolName) != "":
		return "tool:" + strings.TrimSpace(memory.ToolName)
	default:
		return "scope:" + string(memory.Scope)
	}
}

// memoryHeading says what a group of memories is about, or, for one about
// nothing in particular, who it is for: a person's own preference must not
// read as a rule for the whole organization.
func memoryHeading(memory *agent.Memory) string {
	if about := memory.About(); about != "" {
		return "About " + about
	}

	switch memory.Scope {
	case agent.MemoryScopeUser:
		return personalMemoryHeading
	case agent.MemoryScopeRole:
		return roleMemoryHeading
	case agent.MemoryScopeAgent:
		return agentMemoryHeading
	default:
		return organizationMemoryHeading
	}
}

func memoryLine(memory *agent.Memory) string {
	return "- [" + string(memory.Kind) + "] " + memoryPromptContent(memory)
}

func outsideMemoryLine(memory *agent.Memory) string {
	line := "- (recorded as " + strings.ToLower(string(memory.Kind)) + ") "
	if about := memory.About(); about != "" {
		line += about + ": "
	}

	return line + memoryPromptContent(memory)
}

func memoryPromptContent(memory *agent.Memory) string {
	content := strings.TrimSpace(memory.Content)
	if utf8.RuneCountInString(content) <= agent.MemoryPromptExcerptChars {
		return content
	}

	return stringutils.Ellipsize(content, agent.MemoryPromptExcerptChars) +
		" (cut short: call recall_memory with id " + memory.ID.String() +
		" to read all of it)"
}

type memoryGroup struct {
	heading  string
	memories []*agent.Memory
}

func groupMemories(memories []*agent.Memory) []memoryGroup {
	groups := make([]memoryGroup, 0, len(memories))
	index := make(map[string]int, len(memories))
	for _, memory := range memories {
		key := memoryGroupKey(memory)
		at, ok := index[key]
		if !ok {
			at = len(groups)
			index[key] = at
			groups = append(groups, memoryGroup{heading: memoryHeading(memory)})
		}
		groups[at].memories = append(groups[at].memories, memory)
	}

	for idx := range groups {
		slices.SortStableFunc(groups[idx].memories, func(a, b *agent.Memory) int {
			return a.Kind.Rank() - b.Kind.Rank()
		})
	}

	return groups
}
