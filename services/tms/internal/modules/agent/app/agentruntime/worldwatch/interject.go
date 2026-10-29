package worldwatch

import (
	"maps"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpagecontext"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	actionDeleted = "deleted"
	actionCreated = "created"
)

const (
	WorldCheckEvery   int64 = 3
	ownWriteGrace     int64 = 15
	maxWatchedRecords       = 64
	minRecordIDLength       = 27
	maxRecordIDLength       = 100
)

type Steer struct {
	ID       pulid.ID                     `json:"id"`
	Content  string                       `json:"content"`
	Mentions []agentpagecontext.EntityRef `json:"mentions,omitempty"`
}

type Check struct {
	Cursor  string   `json:"cursor"`
	Records []string `json:"records"`
}

type Interjections struct {
	Steers []Steer
	World  *realtimeports.RecordChanges
}

type OwnWrite struct {
	ID string `json:"id"`
	At int64  `json:"at"`
}

type State struct {
	Cursor    string                       `json:"cursor"`
	Records   []agentpagecontext.EntityRef `json:"records,omitempty"`
	Own       []OwnWrite                   `json:"own,omitempty"`
	CheckedAt int64                        `json:"checkedAt,omitempty"`
}

func WatchedRecords(req *agentruntimeports.RunRequest) []agentpagecontext.EntityRef {
	if len(req.Context.Anchors) == 0 {
		return req.Records
	}
	records := make([]agentpagecontext.EntityRef, 0, len(req.Records)+len(req.Context.Anchors))
	records = append(records, req.Records...)
	for _, anchor := range req.Context.Anchors {
		if anchor.Note == "" {
			records = append(
				records,
				agentpagecontext.EntityRef{Type: anchor.Kind, ID: anchor.ID, Label: anchor.Label},
			)
		}
	}

	return records
}

func OpenWorld(cursor string, records []agentpagecontext.EntityRef, now int64) *State {
	if cursor == "" {
		return nil
	}
	world := &State{
		Cursor:    cursor,
		Records:   make([]agentpagecontext.EntityRef, 0, min(len(records)+4, maxWatchedRecords)),
		CheckedAt: now,
	}
	for _, record := range records {
		world.Watch(record)
	}

	return world
}

func (w *State) Clone() *State {
	if w == nil {
		return nil
	}

	return &State{
		Cursor:    w.Cursor,
		Records:   slices.Clone(w.Records),
		Own:       slices.Clone(w.Own),
		CheckedAt: w.CheckedAt,
	}
}

func (w *State) Watch(record agentpagecontext.EntityRef) {
	if w == nil || !recordID(record.ID) || len(w.Records) >= maxWatchedRecords {
		return
	}
	idx := slices.IndexFunc(w.Records, func(known agentpagecontext.EntityRef) bool {
		return known.ID == record.ID
	})
	if idx < 0 {
		w.Records = append(w.Records, record)
		return
	}
	if w.Records[idx].Label == "" && record.Label != "" {
		w.Records[idx].Label = record.Label
	}
	if w.Records[idx].Type == "" && record.Type != "" {
		w.Records[idx].Type = record.Type
	}
}

func (w *State) Read(call *llmports.ToolCall, summary string) {
	if w == nil {
		return
	}
	for _, record := range RecordsRead(call.Name, call.Arguments, summary) {
		w.Watch(record)
	}
}

func RecordsRead(
	name string,
	arguments map[string]any,
	summary string,
) []agentpagecontext.EntityRef {
	name = agenttoolports.ReadingOrder(name)
	if !strings.HasPrefix(name, "get_") || len(arguments) == 0 {
		return nil
	}
	kind := strings.TrimPrefix(name, "get_")
	records := make([]agentpagecontext.EntityRef, 0, 1)
	for _, key := range slices.Sorted(maps.Keys(arguments)) {
		if !strings.HasSuffix(key, "Id") && !strings.HasSuffix(key, "ID") && key != "id" {
			continue
		}
		id, ok := arguments[key].(string)
		if !ok || !recordID(id) {
			continue
		}
		records = append(records, agentpagecontext.EntityRef{Type: kind, ID: id, Label: summary})
	}

	return records
}

func (w *State) Wrote(action *agentruntimeports.PendingAction, at int64) {
	if w == nil || action == nil || !action.Executed || action.Target == nil ||
		action.Target.ID.IsNil() {
		return
	}
	id := action.Target.ID.String()
	w.Watch(agentpagecontext.EntityRef{Type: string(action.Target.Resource), ID: id})
	idx := slices.IndexFunc(w.Own, func(own OwnWrite) bool { return own.ID == id })
	if idx < 0 {
		w.Own = append(w.Own, OwnWrite{ID: id, At: at})
		return
	}
	w.Own[idx].At = at
}

func (w *State) Due(now int64) *Check {
	if w == nil || w.Cursor == "" || len(w.Records) == 0 || now-w.CheckedAt < WorldCheckEvery {
		return nil
	}
	ids := make([]string, 0, len(w.Records))
	for _, record := range w.Records {
		ids = append(ids, record.ID)
	}

	return &Check{Cursor: w.Cursor, Records: ids}
}

func (w *State) News(
	found *realtimeports.RecordChanges,
	now int64,
) []realtimeports.WatchedRecordChange {
	w.CheckedAt = now
	if found == nil {
		return nil
	}
	if found.Cursor != "" {
		w.Cursor = found.Cursor
	}
	if len(found.Changes) == 0 {
		return nil
	}

	news := make([]realtimeports.WatchedRecordChange, 0, len(found.Changes))
	for i := range found.Changes {
		change := &found.Changes[i]
		if w.ownChange(change) {
			continue
		}
		idx := slices.IndexFunc(w.Records, func(record agentpagecontext.EntityRef) bool {
			return record.ID == change.RecordID
		})
		if idx < 0 {
			continue
		}
		change.Label = watchedName(w.Records[idx])
		news = append(news, *change)
	}

	return news
}

func (w *State) ownChange(change *realtimeports.WatchedRecordChange) bool {
	idx := slices.IndexFunc(w.Own, func(own OwnWrite) bool { return own.ID == change.RecordID })

	return idx >= 0 && change.At <= w.Own[idx].At+ownWriteGrace
}

func recordID(id string) bool {
	if len(id) < minRecordIDLength || len(id) > maxRecordIDLength ||
		!strings.Contains(id, "_") {
		return false
	}
	for _, r := range id {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '_' {
			return false
		}
	}

	return true
}

func watchedName(record agentpagecontext.EntityRef) string {
	kind := strings.ReplaceAll(record.Type, "_", " ")
	switch {
	case record.Label != "" && kind != "" && !strings.Contains(
		strings.ToLower(record.Label), strings.ToLower(kind),
	):
		return kind + " " + record.Label
	case record.Label != "":
		return record.Label
	case kind != "":
		return kind + " " + record.ID
	default:
		return record.ID
	}
}

const SteerNotice = "[Notice from the system, not from the person: the person added the " +
	"message above while you were still working. Take it into account from here: change " +
	"course if it calls for that, keep what still stands, and do not start over.]"

func SteerPrompt(content string, mentions []agentpagecontext.EntityRef) string {
	var b strings.Builder
	b.Grow(len(content) + len(mentions)*48)
	b.WriteString(content)
	if len(mentions) > 0 {
		b.WriteString("\n\nRecords they named:")
		for _, mention := range mentions {
			b.WriteString("\n- ")
			b.WriteString(watchedName(mention))
			if mention.Label != "" {
				b.WriteString(" (id ")
				b.WriteString(mention.ID)
				b.WriteString(")")
			}
		}
	}

	return b.String()
}

const ChangePreamble = "[Notice from the system, not from the person: records you are " +
	"working with changed while you were working.]"

const worldChangeClosing = "What you read about them earlier may be out of date. Read each " +
	"one again before you rely on it or change it, and tell the person if it changes your answer."

func ChangeNotice(
	changes []realtimeports.WatchedRecordChange,
	actor *permissionports.RequestActor,
) string {
	var b strings.Builder
	b.WriteString(ChangePreamble)
	for i := range changes {
		change := &changes[i]
		b.WriteString("\n- ")
		b.WriteString(change.Label)
		if change.Label != change.RecordID {
			b.WriteString(" (")
			b.WriteString(change.RecordID)
			b.WriteString(")")
		}
		b.WriteString(" was ")
		b.WriteString(changeVerb(change.Action))
		b.WriteString(" ")
		b.WriteString(changedBy(change, actor))
		if len(change.Fields) > 0 && change.Action != actionDeleted {
			b.WriteString(": ")
			b.WriteString(strings.Join(change.Fields, ", "))
		}
		b.WriteString(".")
	}
	b.WriteString("\n")
	b.WriteString(worldChangeClosing)

	return b.String()
}

func changeVerb(action string) string {
	switch verb := strings.TrimPrefix(action, "bulk_"); verb {
	case actionDeleted, actionCreated:
		return verb
	default:
		return "changed"
	}
}

func changedBy(
	change *realtimeports.WatchedRecordChange,
	actor *permissionports.RequestActor,
) string {
	if actor != nil && actor.UserID.IsNotNil() && change.ActorUserID == actor.UserID.String() &&
		change.ActorType != string(permissionports.PrincipalTypeAgent) {
		return "by the person you are working for, outside this conversation"
	}
	switch kernelports.PrincipalType(change.ActorType) {
	case permissionports.PrincipalTypeAgent:
		return "by an agent"
	case permissionports.PrincipalTypeSystem:
		return "by the system"
	case permissionports.PrincipalTypeAPIKey:
		return "through the API"
	case permissionports.PrincipalTypeUser:
		return "by someone else"
	default:
		return "elsewhere"
	}
}
