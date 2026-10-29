package agentquerytoolservice

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/domain/extensionwebwatch"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/toolschema"
)

const (
	webWatchListLimit = 25
	webWatchDayLayout = "2006-01-02"
	webWatchesNote    = "Stop a watch the person no longer needs with web_stop_watch. What a " +
		"watch finds comes back to the work that set it; set a new one with web_watch."
	noWebWatchesNote = "There are no web watches set for this person. Set one with web_watch " +
		"when they ask to be told when something happens."
)

type listWebWatchesTool struct {
	watches agentextensionports.WebWatcher
}

func newListWebWatchesTool(watches agentextensionports.WebWatcher) agenttoolports.AgentQueryTool {
	return &listWebWatchesTool{watches: watches}
}

func (t *listWebWatchesTool) Name() string { return agentextension.ToolListWebWatches }

func (t *listWebWatchesTool) Description() string {
	return "List the web watches set for the person: what each watches for, how often and " +
		"until when, how often it found something new, and whether it still runs. Call it " +
		"before web_stop_watch to find a watch's id, or to answer what is being watched."
}

func (t *listWebWatchesTool) ParamSchema() map[string]any {
	return map[string]any{
		toolschema.KeyType: "object",
		toolschema.KeyProperties: map[string]any{
			"includeEnded": map[string]any{
				toolschema.KeyType: "boolean",
				toolschema.KeyDescription: "Also list watches that ended, were stopped or " +
					"failed. Off unless asked about past watches.",
			},
		},
		toolschema.KeyAdditionalProperties: false,
	}
}

func (t *listWebWatchesTool) Policy() agenttoolports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{
		resource: permission.ResourceWebResearch,
		rationale: "Lists the web watches set for the person; it changes nothing, reaches " +
			"no one, and returns only what the agents wrote when they set them.",
	})
}

func (t *listWebWatchesTool) SearchTerms() []string {
	return []string{
		"web watches", "what am i watching", "monitors", "alerts", "stop watching",
		"watch list",
	}
}

type webWatchEntry struct {
	ID           string   `json:"id"`
	WatchingFor  string   `json:"watchingFor"`
	Search       string   `json:"search"`
	Sites        []string `json:"sites,omitempty"`
	HowOften     string   `json:"howOften"`
	Status       string   `json:"status"`
	EndsOn       string   `json:"endsOn"`
	TimesFound   int      `json:"timesFound"`
	LastChecked  string   `json:"lastChecked,omitempty"`
	Problem      string   `json:"problem,omitempty"`
	FromThisChat bool     `json:"fromThisConversation"`
}

type webWatchesAnswer struct {
	Watches []webWatchEntry `json:"watches"`
	Note    string          `json:"note"`
}

func (t *listWebWatchesTool) Query(
	ctx context.Context,
	params *agenttoolports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	req := &agentextensionports.ListWebWatchesRequest{
		TenantInfo: params.Actor.TenantInfo(),
		Status:     extensionwebwatch.WebWatchStatusActive,
		Limit:      webWatchListLimit,
	}
	if includeEnded, _ := params.Params["includeEnded"].(bool); includeEnded {
		req.Status = ""
	}
	if person := params.Actor.PersonUserID(); person.IsNotNil() {
		req.RequestedBy = person
	} else {
		req.DefinitionID = params.AgentDefinitionID
	}

	listed, err := t.watches.ListWebWatches(ctx, req)
	if err != nil {
		return nil, err
	}

	entries := make([]webWatchEntry, 0, len(listed.Items))
	for idx := range listed.Items {
		watch := &listed.Items[idx]
		entry := webWatchEntry{
			ID:           watch.ID.String(),
			WatchingFor:  watch.Purpose,
			Search:       watch.Query,
			Sites:        watch.Sites,
			HowOften:     watch.Cadence.Words(),
			Status:       watch.Status.String(),
			EndsOn:       watchDay(watch.EndsAt),
			TimesFound:   watch.FindingCount,
			Problem:      watch.Failure,
			FromThisChat: params.ThreadID.IsNotNil() && watch.ThreadID == params.ThreadID,
		}
		if watch.LastCheckedAt != nil {
			entry.LastChecked = watchDay(*watch.LastCheckedAt)
		}
		entries = append(entries, entry)
	}

	answer := webWatchesAnswer{Watches: entries, Note: webWatchesNote}
	if len(entries) == 0 {
		answer.Note = noWebWatchesNote
	}

	return answer, nil
}

func watchDay(at int64) string {
	return time.Unix(at, 0).UTC().Format(webWatchDayLayout)
}
