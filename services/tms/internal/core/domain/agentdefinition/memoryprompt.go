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

const organizationMemoryHeading = "For the whole organization"

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

func (d *Definition) FitMemories(rc *RuntimeContext) []*agent.Memory {
	if rc == nil || len(rc.Memories) == 0 {
		return nil
	}

	ordered := orderMemoriesForPrompt(rc)
	budget := d.EffectiveMemoryTokenBudget()
	spent := 0
	headed := make(map[string]struct{}, len(ordered))
	fitted := make([]*agent.Memory, 0, len(ordered))
	for _, memory := range ordered {
		if memory.DrawnFromOutside() {
			cost := llmtokens.Estimate(outsideMemoryLine(memory))
			if spent+cost <= budget {
				spent += cost
				fitted = append(fitted, memory)
			}

			continue
		}

		cost := llmtokens.Estimate(memoryLine(memory))
		key := memoryGroupKey(memory)
		_, hasHeading := headed[key]
		if !hasHeading {
			cost += llmtokens.Estimate(memoryHeading(memory))
		}
		if spent+cost > budget {
			continue
		}
		spent += cost
		headed[key] = struct{}{}
		fitted = append(fitted, memory)
	}

	return fitted
}

type rankedMemory struct {
	memory *agent.Memory
	tier   memoryTier
	rank   int
}

func orderMemoriesForPrompt(rc *RuntimeContext) []*agent.Memory {
	relations := memoryRelations(rc.MemorySubjects)
	loaded := loadedToolNames(rc)

	ranked := make([]rankedMemory, 0, len(rc.Memories))
	for idx, memory := range rc.Memories {
		if memory == nil || strings.TrimSpace(memory.Content) == "" {
			continue
		}
		ranked = append(ranked, rankedMemory{
			memory: memory,
			tier:   tierOf(memory, relations, loaded),
			rank:   idx,
		})
	}

	slices.SortStableFunc(ranked, func(a, b rankedMemory) int {
		if a.tier != b.tier {
			return int(a.tier) - int(b.tier)
		}
		if a.tier != memoryTierRest {
			if byKind := a.memory.Kind.Rank() - b.memory.Kind.Rank(); byKind != 0 {
				return byKind
			}
		}

		return a.rank - b.rank
	})

	ordered := make([]*agent.Memory, 0, len(ranked))
	for _, entry := range ranked {
		ordered = append(ordered, entry.memory)
	}

	return ordered
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
	if memory.Kind == agent.MemoryKindInstruction {
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
		return ""
	}
}

func memoryHeading(memory *agent.Memory) string {
	if about := memory.About(); about != "" {
		return "About " + about
	}

	return organizationMemoryHeading
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
