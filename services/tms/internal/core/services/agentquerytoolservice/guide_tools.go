package agentquerytoolservice

import (
	"context"
	"sort"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/productguide"
	"github.com/emoss08/trenova/shared/stringutils"
)

/*
Trenova describing itself, to the person asking.

An agent knew the data but not the product: asked "how do I add a rate
matrix?" it had nothing to answer from, and asked to do something it held no
tool for it said so and stopped. These two read the product guide generated
from the app itself — every page, where it sits, what it needs, and the steps
of what people do there — and take the person to a page when they ask.

Both answer for the person in the conversation and nobody else: which pages
they may open decides the answer, and moving the app only means something
while somebody is looking at it. So both are self-scoped, and a run nobody is
watching holds neither.
*/

const (
	guideAnswerLimit = 5
	createAction     = "create"
)

func guideTenant(params serviceports.QueryToolParams) pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  params.OrganizationID,
		BuID:   params.BusinessUnitID,
		UserID: params.Actor.UserID,
	}
}

type findInTrenovaTool struct {
	guide serviceports.ProductGuide
}

func newFindInTrenovaTool(guide serviceports.ProductGuide) serviceports.AgentQueryTool {
	return &findInTrenovaTool{guide: guide}
}

func (t *findInTrenovaTool) Name() string { return "find_in_trenova" }

func (t *findInTrenovaTool) Description() string {
	return "Find where something is in Trenova and how to do it there. It returns the page, " +
		"where it sits in the menu, its link, whether this person may open it, and the steps " +
		"with the exact labels on screen. Use it for \"how do I…\", " +
		"\"where is…\" and \"what is this page for\", and whenever the person asks for " +
		"something you hold no tool for, so you can tell them where they do it themselves. " +
		"Answer from what it returns; never invent a page, a path, a button or a step."
}

func (t *findInTrenovaTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"question": map[string]any{
				"type":        "string",
				"description": "What the person wants to find or do, in their words.",
			},
			"page": map[string]any{
				"type": "string",
				"description": "Optional: a page's path, to answer from that page's own " +
					"guide, such as the page the person is on.",
			},
		},
		"required":             []string{"question"},
		"additionalProperties": false,
	}
}

func (t *findInTrenovaTool) Policy() serviceports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{
		resource: permission.ResourceAssistant,
		scope:    agent.ToolScopeSelf,
		effect:   agent.ToolEffectDiscover,
		rationale: "Searches the product guide for the caller; nothing changes and nothing is " +
			"sent.",
	})
}

func (t *findInTrenovaTool) SearchTerms() []string {
	return []string{"how do i", "where is", "help", "guide", "page", "menu", "navigate", "find"}
}

type guideTask struct {
	Title string   `json:"title"`
	Steps []string `json:"steps,omitempty"`
}

type guideAnswer struct {
	Name     string     `json:"name"`
	Location string     `json:"location"`
	Path     string     `json:"path"`
	Summary  string     `json:"summary,omitempty"`
	CanOpen  bool       `json:"canOpen"`
	Missing  []string   `json:"missing,omitempty"`
	Task     *guideTask `json:"task,omitempty"`
	Related  []string   `json:"related,omitempty"`
}

type guideSearchResult struct {
	Answers []guideAnswer `json:"answers"`
	Note    string        `json:"note"`
}

func (t *findInTrenovaTool) Query(
	ctx context.Context,
	params serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	question, err := requireString(params.Params, "question")
	if err != nil {
		return nil, err
	}

	matches, err := t.guide.Search(ctx, &serviceports.ProductGuideSearchRequest{
		Actor:      params.Actor,
		TenantInfo: guideTenant(params),
		Query:      question,
		Page:       optionalString(params.Params, "page"),
		Limit:      guideAnswerLimit,
		Attribution: serviceports.AIUsageAttribution{
			UserID:            params.Actor.UserID,
			AgentDefinitionID: params.AgentDefinitionID,
		},
	})
	if err != nil {
		return nil, err
	}

	if len(matches) == 0 {
		return guideSearchResult{
			Answers: []guideAnswer{},
			Note: "Nothing in the guide matches. Ask the person to say it another way, or say " +
				"you do not know where that is; do not guess a page.",
		}, nil
	}

	answers := make([]guideAnswer, 0, len(matches))
	for i := range matches {
		answers = append(answers, answerFrom(&matches[i], i == 0))
	}

	return guideSearchResult{
		Answers: answers,
		Note: "Link a page as [name](path) with the path given. Quote step labels exactly. " +
			"Where canOpen is false, say what access is missing instead of sending them there. " +
			"Call open_page only if the person asks to be taken there.",
	}, nil
}

// answerFrom shapes one match. Only the best one carries its steps: the
// others are alternatives the person may mean, and their steps would crowd
// the answer out of the model's context.
func answerFrom(match *serviceports.ProductGuideMatch, withSteps bool) guideAnswer {
	page := match.Page
	answer := guideAnswer{
		Name:     page.Name,
		Location: page.Location(),
		Path:     page.Path,
		Summary:  stringutils.FirstNonEmpty(page.Summary, page.Description),
		CanOpen:  match.CanOpen,
		Missing:  match.Missing,
	}
	if match.Task != nil {
		answer.Task = &guideTask{Title: match.Task.Title}
		if withSteps {
			answer.Task.Steps = match.Task.Steps
		}
	}
	if withSteps {
		answer.Related = page.Related
	}

	return answer
}

type openPageTool struct {
	guide    serviceports.ProductGuide
	entities []string
}

func newOpenPageTool(guide serviceports.ProductGuide) serviceports.AgentQueryTool {
	entities := make([]string, 0, len(productguide.Default.Records))
	for _, record := range productguide.Default.Records {
		entities = append(entities, record.Entity)
	}
	sort.Strings(entities)

	return &openPageTool{guide: guide, entities: entities}
}

func (t *openPageTool) Name() string { return "open_page" }

func (t *openPageTool) Description() string {
	return "Take the person to a page or a record in Trenova: the app moves there as soon as " +
		"you call it. Use it only when they ask to be taken, opened or shown somewhere. Pass " +
		"a page's path from find_in_trenova, or a record's entity and id from the tool that " +
		"found it; action create opens the page's create form. It refuses a page they may " +
		"not open."
}

func (t *openPageTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"page": map[string]any{
				"type":        "string",
				"description": "A page's path, exactly as find_in_trenova returned it.",
			},
			"entity": map[string]any{
				"type":        "string",
				"enum":        t.entities,
				"description": "The kind of record to open, with recordId, instead of a page.",
			},
			"recordId": map[string]any{
				"type": "string",
				"description": "The record's id, from the tool that found it (get_shipment, " +
					"list_invoices and the like) or the record on screen.",
			},
			"action": map[string]any{
				"type":        "string",
				"enum":        []string{createAction},
				"description": "Optional: create opens the page's create form.",
			},
		},
		"additionalProperties": false,
	}
}

func (t *openPageTool) Policy() serviceports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{
		resource:  permission.ResourceAssistant,
		scope:     agent.ToolScopeSelf,
		effect:    agent.ToolEffectNavigate,
		rationale: "Opens a page in the caller's own browser; nothing changes and nothing is sent.",
	})
}

func (t *openPageTool) SearchTerms() []string {
	return []string{"take me", "go to", "open", "navigate", "show me the page", "bring up"}
}

// NavigationResult is what open_page returns, and what the navigation
// artifact is built from.
type NavigationResult struct {
	Path     string `json:"path"`
	Name     string `json:"name"`
	Location string `json:"location"`
	Page     string `json:"page"`
	Note     string `json:"note"`
}

func (t *openPageTool) Query(
	ctx context.Context,
	params serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	destination, err := t.guide.Destination(ctx, &serviceports.ProductGuideDestinationRequest{
		Actor:      params.Actor,
		TenantInfo: guideTenant(params),
		Page:       optionalString(params.Params, "page"),
		Entity:     optionalString(params.Params, "entity"),
		RecordID:   optionalString(params.Params, "recordId"),
		Create:     optionalString(params.Params, "action") == createAction,
	})
	if err != nil {
		return nil, err
	}

	return NavigationResult{
		Path:     destination.Path,
		Name:     destination.Label,
		Location: destination.Page.Location(),
		Page:     destination.Page.Path,
		Note: "The app is moving there now. Say where you took them in one short sentence; " +
			"do not repeat the link.",
	}, nil
}
