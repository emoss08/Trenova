package agentruntime

import (
	"context"
	"maps"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const ChangeInterjections = "agent-loop-interjections"

const (
	actionDeleted = "deleted"
	actionCreated = "created"
)

const (
	worldCheckEvery   int64 = 3
	ownWriteGrace     int64 = 15
	maxWatchedRecords       = 64
	minRecordIDLength       = 27
	maxRecordIDLength       = 100
)

type Steer struct {
	ID       pulid.ID          `json:"id"`
	Content  string            `json:"content"`
	Mentions []agent.EntityRef `json:"mentions,omitempty"`
}

type WorldCheck struct {
	Cursor  string   `json:"cursor"`
	Records []string `json:"records"`
}

type Interjections struct {
	Steers []Steer
	World  *serviceports.RecordChanges
}

type OwnWrite struct {
	ID string `json:"id"`
	At int64  `json:"at"`
}

type WorldState struct {
	Cursor    string            `json:"cursor"`
	Records   []agent.EntityRef `json:"records,omitempty"`
	Own       []OwnWrite        `json:"own,omitempty"`
	CheckedAt int64             `json:"checkedAt,omitempty"`
}

func (s *Service) watchWorld(
	ctx context.Context,
	req *serviceports.RunRequest,
	now int64,
) *WorldState {
	if s.changes == nil || req.Delegation != nil || req.Actor == nil {
		return nil
	}
	cursor, err := s.changes.Head(ctx, req.Actor.OrganizationID, req.Actor.BusinessUnitID)
	if err != nil {
		s.logger.Debug("the turn will not watch its records for changes", zap.Error(err))
		return nil
	}

	return openWorld(cursor, watchedRecords(req), now)
}

func watchedRecords(req *serviceports.RunRequest) []agent.EntityRef {
	if len(req.Context.Anchors) == 0 {
		return req.Records
	}
	records := make([]agent.EntityRef, 0, len(req.Records)+len(req.Context.Anchors))
	records = append(records, req.Records...)
	for _, anchor := range req.Context.Anchors {
		if anchor.Note == "" {
			records = append(records, agent.EntityRef{Type: anchor.Kind, ID: anchor.ID, Label: anchor.Label})
		}
	}

	return records
}

func openWorld(cursor string, records []agent.EntityRef, now int64) *WorldState {
	if cursor == "" {
		return nil
	}
	world := &WorldState{
		Cursor:    cursor,
		Records:   make([]agent.EntityRef, 0, min(len(records)+4, maxWatchedRecords)),
		CheckedAt: now,
	}
	for _, record := range records {
		world.watch(record)
	}

	return world
}

func (w *WorldState) clone() *WorldState {
	if w == nil {
		return nil
	}

	return &WorldState{
		Cursor:    w.Cursor,
		Records:   slices.Clone(w.Records),
		Own:       slices.Clone(w.Own),
		CheckedAt: w.CheckedAt,
	}
}

func (w *WorldState) watch(record agent.EntityRef) {
	if w == nil || !recordID(record.ID) || len(w.Records) >= maxWatchedRecords {
		return
	}
	idx := slices.IndexFunc(w.Records, func(known agent.EntityRef) bool {
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

func (w *WorldState) read(call *serviceports.ToolCall, summary string) {
	if w == nil {
		return
	}
	for _, record := range RecordsRead(call.Name, call.Arguments, summary) {
		w.watch(record)
	}
}

func RecordsRead(name string, arguments map[string]any, summary string) []agent.EntityRef {
	if !strings.HasPrefix(name, "get_") || len(arguments) == 0 {
		return nil
	}
	kind := strings.TrimPrefix(name, "get_")
	records := make([]agent.EntityRef, 0, 1)
	for _, key := range slices.Sorted(maps.Keys(arguments)) {
		if !strings.HasSuffix(key, "Id") && !strings.HasSuffix(key, "ID") && key != "id" {
			continue
		}
		id, ok := arguments[key].(string)
		if !ok || !recordID(id) {
			continue
		}
		records = append(records, agent.EntityRef{Type: kind, ID: id, Label: summary})
	}

	return records
}

func (w *WorldState) wrote(action *serviceports.PendingAction, at int64) {
	if w == nil || action == nil || !action.Executed || action.Target == nil ||
		action.Target.ID.IsNil() {
		return
	}
	id := action.Target.ID.String()
	w.watch(agent.EntityRef{Type: string(action.Target.Resource), ID: id})
	idx := slices.IndexFunc(w.Own, func(own OwnWrite) bool { return own.ID == id })
	if idx < 0 {
		w.Own = append(w.Own, OwnWrite{ID: id, At: at})
		return
	}
	w.Own[idx].At = at
}

func (t *Turn) noteWorld(fx TurnEffects, call *serviceports.ToolCall, outcome *toolOutcome) {
	if t.world == nil {
		return
	}
	t.world.read(call, outcome.summary)
	if outcome.action != nil && outcome.action.Executed {
		t.world.wrote(outcome.action, fx.Now())
	}
}

func (w *WorldState) due(now int64) *WorldCheck {
	if w == nil || w.Cursor == "" || len(w.Records) == 0 || now-w.CheckedAt < worldCheckEvery {
		return nil
	}
	ids := make([]string, 0, len(w.Records))
	for _, record := range w.Records {
		ids = append(ids, record.ID)
	}

	return &WorldCheck{Cursor: w.Cursor, Records: ids}
}

func (w *WorldState) news(
	found *serviceports.RecordChanges,
	now int64,
) []serviceports.WatchedRecordChange {
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

	news := make([]serviceports.WatchedRecordChange, 0, len(found.Changes))
	for i := range found.Changes {
		change := &found.Changes[i]
		if w.ownChange(change) {
			continue
		}
		idx := slices.IndexFunc(w.Records, func(record agent.EntityRef) bool {
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

func (w *WorldState) ownChange(change *serviceports.WatchedRecordChange) bool {
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

func watchedName(record agent.EntityRef) string {
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

func (s *Service) interject(t *Turn, fx TurnEffects) {
	if t.req.Delegation != nil {
		return
	}
	var now int64
	var check *WorldCheck
	if t.world != nil {
		now = fx.Now()
		check = t.world.due(now)
	}
	got := fx.Interject(t, check)
	for _, steer := range got.Steers {
		t.steer(fx, steer)
	}
	if check == nil {
		return
	}
	if changes := t.world.news(got.World, now); len(changes) > 0 {
		t.worldChanged(fx, changes)
	}
}

func (t *Turn) steer(fx TurnEffects, steer Steer) {
	content := strings.TrimSpace(steer.Content)
	if content == "" {
		return
	}
	mentions := agent.NormalizeEntityRefs(steer.Mentions)
	for _, mention := range mentions {
		t.world.watch(mention)
	}

	t.messages = append(t.messages, serviceports.Message{
		Role:    serviceports.RoleUser,
		Content: SteerPrompt(content, mentions),
	})
	t.result.Messages = append(t.result.Messages, conversation.Message{
		Role:      conversation.RoleUser,
		Kind:      conversation.MessageKindSteer,
		Content:   content,
		Mentions:  mentions,
		CreatedAt: fx.Now(),
	})
	fx.Emit(serviceports.StreamEvent{
		Event: serviceports.AssistantEventSteered,
		Data: serviceports.AssistantSteeredEvent{
			ID:       steer.ID,
			Content:  content,
			Mentions: mentions,
		},
	})
}

func (t *Turn) worldChanged(fx TurnEffects, changes []serviceports.WatchedRecordChange) {
	notice := WorldChangeNotice(changes, t.req.Actor)
	clear(t.repeats.reads)

	t.messages = append(t.messages, serviceports.Message{
		Role:    serviceports.RoleUser,
		Content: notice,
	})
	t.result.Messages = append(t.result.Messages, conversation.Message{
		Role:         conversation.RoleUser,
		Kind:         conversation.MessageKindWorldChange,
		Content:      notice,
		WorldChanges: changes,
		CreatedAt:    fx.Now(),
	})
	fx.Emit(serviceports.StreamEvent{
		Event: serviceports.AssistantEventWorldChanged,
		Data:  serviceports.AssistantWorldChangedEvent{Changes: changes},
	})
}

const steerPreamble = "[The person added this while you were still working. Take it into " +
	"account from here: change course if it calls for that, keep what still stands, and do " +
	"not start over.]\n\n"

func SteerPrompt(content string, mentions []agent.EntityRef) string {
	var b strings.Builder
	b.Grow(len(steerPreamble) + len(content) + len(mentions)*48)
	b.WriteString(steerPreamble)
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

const worldChangePreamble = "[Notice from the system, not from the person: records you are " +
	"working with changed while you were working.]"

const worldChangeClosing = "What you read about them earlier may be out of date. Read each " +
	"one again before you rely on it or change it, and tell the person if it changes your answer."

func WorldChangeNotice(
	changes []serviceports.WatchedRecordChange,
	actor *serviceports.RequestActor,
) string {
	var b strings.Builder
	b.WriteString(worldChangePreamble)
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
	change *serviceports.WatchedRecordChange,
	actor *serviceports.RequestActor,
) string {
	if actor != nil && actor.UserID.IsNotNil() && change.ActorUserID == actor.UserID.String() &&
		change.ActorType != string(serviceports.PrincipalTypeAgent) {
		return "by the person you are working for, outside this conversation"
	}
	switch serviceports.PrincipalType(change.ActorType) {
	case serviceports.PrincipalTypeAgent:
		return "by an agent"
	case serviceports.PrincipalTypeSystem:
		return "by the system"
	case serviceports.PrincipalTypeAPIKey:
		return "through the API"
	case serviceports.PrincipalTypeUser:
		return "by someone else"
	default:
		return "elsewhere"
	}
}
