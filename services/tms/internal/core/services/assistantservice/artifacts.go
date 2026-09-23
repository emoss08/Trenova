package assistantservice

import (
	"context"
	"errors"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"go.uber.org/zap"
)

// The tools whose results a person would rather see than read. A preview is
// a table; a run is a card with a download; a get is the record itself. The
// mapper reads each result in its JSON form, exactly as the model does, so
// it depends on the tool's contract rather than on its Go types.
const (
	toolPreviewReport = "preview_report"
	toolRunReport     = "run_report"
	toolGetReportRun  = "get_report_run"
	getToolPrefix     = "get_"
	listToolPrefix    = "list_"
	searchToolPrefix  = "search_"
	toolComposeView   = "compose_table_view"
	toolExplainRate   = "explain_rate"
	toolCompareRuns   = "compare_report_runs"

	maxArtifactTitleRunes = 120
	minPreviewRows        = 1
)

// draftSpec names how an outbound message proposal reads as a draft: which
// parameters are the subject, the body and the recipients, and what the
// draft is called when it has no subject of its own.
type draftSpec struct {
	title      string
	subjectKey string
	bodyKey    string
	toKey      string
}

// draftSpecs are the tools whose proposal is a message waiting to go. The
// draft is a view over the proposal: approving it is what sends.
var draftSpecs = map[string]draftSpec{
	"email_customer":        {title: "Customer update", subjectKey: "subject", bodyKey: "body"},
	"send_detention_notice": {title: "Detention notice"},
	// The reply's subject and recipient come from the message it answers,
	// not from the proposal, so the draft carries the body alone.
	"reply_to_inbound_message": {title: "Inbox reply", bodyKey: "body"},
	"request_missing_docs": {
		title:      "Document request",
		subjectKey: "subject",
		bodyKey:    "body",
		toKey:      "to",
	},
}

// recordLabelKeys are tried in order for the words that name a record on
// its card: a shipment by its PRO, an invoice by its number, a person by
// their name, a unit by its number.
var recordLabelKeys = []string{
	"proNumber",
	"invoiceNumber",
	"referenceNumber",
	"name",
	"code",
	"unitNumber",
	"licenseNumber",
}

// artifactRecorder collects what one turn produced. It saves each artifact
// as the tool that made it finishes, so the pane can open it while the reply
// is still arriving, and ties them to their messages once the turn is saved.
type artifactRecorder struct {
	repo     repositories.AssistantArtifactRepository
	activity services.AgentActivityPublisher
	logger   *zap.Logger
	ctx      context.Context
	thread   *conversation.Thread
	tenant   pagination.TenantInfo
	actor    services.AuditActor
	emit     services.AssistantStreamEmitter
	recorded []*assistantartifact.Artifact
}

// newArtifactRecorder returns nil when there is nowhere to keep artifacts,
// and every method on a nil recorder is a no-op: a turn without a pane is
// still a turn.
func (s *Service) newArtifactRecorder(
	ctx context.Context,
	thread *conversation.Thread,
	tenant pagination.TenantInfo,
	actor *services.RequestActor,
	emit services.AssistantStreamEmitter,
) *artifactRecorder {
	if s.artifacts == nil {
		return nil
	}
	if emit == nil {
		emit = func(services.StreamEvent) {}
	}

	return &artifactRecorder{
		repo:     s.artifacts,
		activity: s.activity,
		logger:   s.logger,
		ctx:      ctx,
		thread:   thread,
		tenant:   tenant,
		actor:    actor.AuditActorOrSystem(),
		emit:     emit,
	}
}

// observer is the hook the runtime calls per tool result; nil when there is
// no recorder, so the runtime skips the call altogether.
func (r *artifactRecorder) observer() services.ToolObserver {
	if r == nil {
		return nil
	}

	return r.observe
}

func (r *artifactRecorder) observe(
	observation services.ToolObservation,
) (*services.ShownArtifact, error) {
	if document, ok := observation.Data.(services.PublishedDocument); ok {
		return r.publish(observation.Call.ID, document)
	}

	artifact := artifactFromObservation(observation)
	if artifact == nil {
		return nil, nil
	}

	saved, err := r.save(artifact)
	if err != nil {
		return nil, nil
	}

	return shownArtifact(saved), nil
}

// publish keeps a document the model wrote. A revision replaces the text of a
// document this conversation already holds, in place, so the pane keeps one
// brief rather than a stack of drafts of it.
func (r *artifactRecorder) publish(
	callID string,
	document services.PublishedDocument,
) (*services.ShownArtifact, error) {
	artifact := documentArtifact(callID, document)

	if !document.ArtifactID.IsNil() {
		existing, err := r.repo.GetByID(r.ctx, repositories.GetArtifactRequest{
			ID:         document.ArtifactID,
			TenantInfo: r.tenant,
		})
		if err != nil || existing.ThreadID != r.thread.ID ||
			existing.Kind != assistantartifact.KindDocument {
			return nil, errUnknownDocument
		}
		artifact.ID = existing.ID
		artifact.MessageID = existing.MessageID
		artifact.SourceToolCallID = existing.SourceToolCallID
		artifact.Pinned = existing.Pinned
	}

	saved, err := r.save(artifact)
	if err != nil {
		return nil, errDocumentNotKept
	}

	return shownArtifact(saved), nil
}

// fromProposals views the turn's recorded proposals as drafts and its plan
// as a checklist, so a person decides them where they can read them.
func (r *artifactRecorder) fromProposals(proposals []*agent.AgentProposal, plan *agent.AgentPlan) {
	if r == nil {
		return
	}

	for _, proposal := range proposals {
		if artifact := draftArtifact(proposal); artifact != nil {
			r.save(artifact)
		}
	}

	if plan != nil {
		r.save(planArtifact(plan, proposals))
	}
}

// attachMessages ties each artifact made by a tool call to the saved
// assistant message that asked for it, once the turn is on disk.
func (r *artifactRecorder) attachMessages(index map[string]pulid.ID) {
	if r == nil {
		return
	}

	for _, artifact := range r.recorded {
		if !artifact.MessageID.IsNil() || artifact.SourceToolCallID == "" {
			continue
		}
		messageID, ok := index[artifact.SourceToolCallID]
		if !ok {
			continue
		}
		artifact.MessageID = messageID
		if _, err := r.repo.Upsert(r.ctx, artifact); err != nil {
			r.logger.Warn("could not tie an artifact to its message",
				zap.String("artifact", artifact.ID.String()),
				zap.Error(err),
			)
		}
	}
}

// artifacts is what the turn produced, for the turn's result.
func (r *artifactRecorder) artifacts() []services.AssistantArtifact {
	if r == nil || len(r.recorded) == 0 {
		return nil
	}

	out := make([]services.AssistantArtifact, 0, len(r.recorded))
	for _, artifact := range r.recorded {
		out = append(out, toAssistantArtifact(artifact))
	}

	return out
}

func (r *artifactRecorder) save(
	artifact *assistantartifact.Artifact,
) (*assistantartifact.Artifact, error) {
	artifact.ThreadID = r.thread.ID
	artifact.OrganizationID = r.tenant.OrgID
	artifact.BusinessUnitID = r.tenant.BuID

	multiErr := errortypes.NewMultiError()
	artifact.Validate(multiErr)
	if multiErr.HasErrors() {
		r.logger.Error("artifact skipped: invalid",
			zap.String("kind", string(artifact.Kind)),
			zap.Error(multiErr),
		)

		return nil, multiErr
	}

	saved, err := r.repo.Upsert(r.ctx, artifact)
	if err != nil {
		r.logger.Error("artifact could not be kept",
			zap.String("thread", r.thread.ID.String()),
			zap.String("kind", string(artifact.Kind)),
			zap.Error(err),
		)

		return nil, err
	}

	r.remember(saved)
	if r.activity != nil {
		r.activity.ArtifactChanged(r.ctx, saved, r.actor, services.ActivityUpdated)
	}
	r.emit(services.StreamEvent{
		Event: services.AssistantEventArtifact,
		Data: services.AssistantArtifactEvent{
			ID:               saved.ID,
			Kind:             saved.Kind,
			Status:           saved.Status,
			Title:            saved.Title,
			SourceToolCallID: saved.SourceToolCallID,
		},
	})

	return saved, nil
}

// remember adds an artifact to what the turn produced, replacing an earlier
// entry for the same artifact: a document revised twice in one turn is one
// artifact, not two.
func (r *artifactRecorder) remember(saved *assistantartifact.Artifact) {
	for idx, recorded := range r.recorded {
		if recorded.ID == saved.ID {
			r.recorded[idx] = saved
			return
		}
	}
	r.recorded = append(r.recorded, saved)
}

// shownArtifact is how the model is told what the person now sees.
func shownArtifact(artifact *assistantartifact.Artifact) *services.ShownArtifact {
	return &services.ShownArtifact{
		ID:    artifact.ID,
		Kind:  string(artifact.Kind),
		Title: artifact.Title,
	}
}

var (
	errUnknownDocument = errors.New(
		"there is no document with that artifactId in this conversation; leave artifactId " +
			"out to publish a new one",
	)
	errDocumentNotKept = errors.New("it could not be saved")
)

// documentArtifact is a write-up the model published, kept as markdown.
func documentArtifact(callID string, document services.PublishedDocument) *assistantartifact.Artifact {
	return &assistantartifact.Artifact{
		Kind:   assistantartifact.KindDocument,
		Status: assistantartifact.StatusReady,
		Title:  artifactTitle(document.Title),
		Payload: map[string]any{
			"format": "markdown",
			"body":   document.Body,
		},
		SourceToolCallID: callID,
	}
}

// artifactFromObservation turns a finished query tool into what the pane
// shows for it, or nil when the result is one to read rather than see.
func artifactFromObservation(observation services.ToolObservation) *assistantartifact.Artifact {
	if observation.Failed || observation.Data == nil || observation.Call.ID == "" {
		return nil
	}

	result, ok := toJSONMap(observation.Data)
	if !ok {
		return nil
	}

	name := observation.Call.Name
	switch {
	case name == toolPreviewReport:
		return previewArtifact(observation.Call.ID, result)
	case name == toolRunReport || name == toolGetReportRun:
		return runArtifact(observation.Call.ID, result)
	case strings.HasPrefix(name, getToolPrefix):
		return entityCardArtifact(observation.Call.ID, name, result)
	case name == toolComposeView:
		return composedViewArtifact(observation.Call.ID, result)
	case name == toolExplainRate:
		return rateArtifact(observation.Call.ID, result)
	case name == toolCompareRuns:
		return runDiffArtifact(observation.Call.ID, result)
	case strings.HasPrefix(name, listToolPrefix), strings.HasPrefix(name, searchToolPrefix):
		return tableArtifact(observation.Call.ID, name, result)
	default:
		return nil
	}
}

// tableArtifact views a list or search result as the table it already is.
//
// This is the common case by a long way — "how many shipments are in transit",
// "which drivers are out of hours" — and it used to produce nothing at all.
// The pane promised a table and then sat empty through the one kind of turn
// that most often earns one, because only a report preview, a report run and
// a single-record get were mapped.
//
// The guard is the shape rather than the name: the list tools share one
// outcome type, but "list" is a common enough verb that a tool named for it
// could return something else entirely. A result without both rows and their
// declared column order is not a table, whatever it is called.
func tableArtifact(callID, toolName string, result map[string]any) *assistantartifact.Artifact {
	rows, ok := result["items"].([]any)
	if !ok || len(rows) == 0 {
		return nil
	}
	columns := stringsOf(result["columns"])
	if len(columns) == 0 {
		return nil
	}

	entity := strings.TrimPrefix(strings.TrimPrefix(toolName, listToolPrefix), searchToolPrefix)
	payload := map[string]any{
		"tool":        toolName,
		"entity":      entity,
		"columns":     columns,
		"rows":        rows,
		"rowCount":    result["count"],
		"searchedFor": stringsOf(result["searchedFor"]),
	}
	fitRows(payload, "rows")

	return &assistantartifact.Artifact{
		Kind:             assistantartifact.KindTableView,
		Status:           assistantartifact.StatusReady,
		Title:            artifactTitle(stringutils.CapitalizeFirst(stringutils.HumanizeSnakeCase(entity))),
		Payload:          payload,
		SourceToolCallID: callID,
	}
}

// composedViewArtifact is a described view as something to open.
//
// The pane shows what it was narrowed to and what could not be, with the link
// to the live table. It is a table_view like a list result, because to the
// reader it is the same thing arrived at a different way — except that this
// one opens rather than being a snapshot.
func composedViewArtifact(callID string, result map[string]any) *assistantartifact.Artifact {
	path := stringOf(result["path"])
	if path == "" {
		return nil
	}

	entity := stringOf(result["entity"])
	payload := map[string]any{
		"entity":      entity,
		"path":        path,
		"explanation": stringOf(result["explanation"]),
		"terms":       stringsOf(result["terms"]),
		"filterCount": result["filterCount"],
		"unresolved":  result["unresolved"],
	}

	return &assistantartifact.Artifact{
		Kind:   assistantartifact.KindTableView,
		Status: assistantartifact.StatusReady,
		Title: artifactTitle(
			stringutils.CapitalizeFirst(stringutils.HumanizeSnakeCase(entity)),
		),
		Payload:          payload,
		SourceToolCallID: callID,
	}
}

// rateArtifact is the ledger behind a price.
//
// A rate explanation read aloud is a wall of figures nobody can check against
// an invoice. As a ledger — every charge with its arithmetic and a running
// total, the limits that bit, what it was priced under — it is the thing a
// person puts next to the invoice line they are disputing.
func rateArtifact(callID string, result map[string]any) *assistantartifact.Artifact {
	// Nothing to explain is a sentence, not a ledger of zeroes.
	if stringOf(result["note"]) != "" {
		return nil
	}
	components, _ := result["components"].([]any)
	if len(components) == 0 {
		return nil
	}

	payload := map[string]any{
		"shipmentId": stringOf(result["shipmentId"]),
		"side":       stringOf(result["side"]),
		"currency":   stringOf(result["currency"]),
		"winner":     result["winner"],
		"tieBreak":   stringOf(result["tieBreak"]),
		"rejected":   result["rejected"],
		"components": components,
		"guardrails": result["guardrails"],
		"totals":     result["totals"],
		"warnings":   stringsOf(result["warnings"]),
	}
	fitRows(payload, "components")

	return &assistantartifact.Artifact{
		Kind:             assistantartifact.KindRateExplanation,
		Status:           assistantartifact.StatusReady,
		Title:            artifactTitle("Rate breakdown"),
		Payload:          payload,
		SourceToolCallID: callID,
	}
}

// runDiffArtifact views a comparison as the table of movements it is.
//
// Read aloud, a diff is a list of numbers with no anchor — "ACME went from
// 2,840.00 to 3,102.50" a dozen times over is not something anybody checks.
// Shown, with the two runs named at the top, the biggest moves first and the
// totals underneath, it is the answer to "what changed since last week".
func runDiffArtifact(callID string, result map[string]any) *assistantartifact.Artifact {
	changes, _ := result["changes"].([]any)
	totals, _ := result["totals"].([]any)

	// A comparison with nothing on either side is a sentence: "nothing moved".
	if len(changes) == 0 && len(totals) == 0 {
		return nil
	}

	payload := map[string]any{
		"before":    result["before"],
		"after":     result["after"],
		"keys":      stringsOf(result["keys"]),
		"measures":  stringsOf(result["measures"]),
		"summary":   result["summary"],
		"changes":   changes,
		"totals":    totals,
		"truncated": result["truncated"],
		"note":      stringOf(result["note"]),
	}
	// The changes are already sorted by risk and then by size, so halving the
	// list drops the smallest movements rather than an arbitrary tail.
	fitRows(payload, "changes")

	return &assistantartifact.Artifact{
		Kind:             assistantartifact.KindRunDiff,
		Status:           assistantartifact.StatusReady,
		Title:            artifactTitle(runDiffTitle(result)),
		Payload:          payload,
		SourceToolCallID: callID,
	}
}

func runDiffTitle(result map[string]any) string {
	side, _ := result["after"].(map[string]any)
	if name := stringOf(side["reportName"]); name != "" {
		return name + " — what changed"
	}

	return "What changed"
}

// stringsOf reads a JSON array of strings, dropping anything that is not one.
func stringsOf(value any) []string {
	raw, ok := value.([]any)
	if !ok {
		return nil
	}

	out := make([]string, 0, len(raw))
	for _, entry := range raw {
		if text, isText := entry.(string); isText && text != "" {
			out = append(out, text)
		}
	}

	return out
}

func previewArtifact(callID string, result map[string]any) *assistantartifact.Artifact {
	rows, _ := result["rows"].([]any)
	payload := map[string]any{
		"name":      stringOf(result["name"]),
		"dataset":   stringOf(result["dataset"]),
		"columns":   result["columns"],
		"rowCount":  result["rowCount"],
		"rows":      rows,
		"totals":    result["totals"],
		"truncated": boolOf(result["truncated"]),
	}
	fitRows(payload, "rows")

	title := stringOf(result["name"])
	if title == "" {
		title = "Preview of " + stringutils.HumanizeSnakeCase(stringOf(result["dataset"]))
	}

	return &assistantartifact.Artifact{
		Kind:             assistantartifact.KindReportPreview,
		Status:           assistantartifact.StatusReady,
		Title:            artifactTitle(title),
		Payload:          payload,
		SourceToolCallID: callID,
	}
}

func runArtifact(callID string, result map[string]any) *assistantartifact.Artifact {
	runID := stringOf(result["runId"])
	if runID == "" {
		return nil
	}

	payload := map[string]any{
		"runId":        runID,
		"reportKey":    stringOf(result["reportKey"]),
		"definitionId": stringOf(result["definitionId"]),
		"reportName":   stringOf(result["reportName"]),
		"status":       stringOf(result["status"]),
		"finished":     boolOf(result["finished"]),
		"format":       stringOf(result["format"]),
		"rowCount":     result["rowCount"],
		"truncated":    boolOf(result["truncated"]),
	}

	title := stringOf(result["reportName"])
	if title == "" {
		title = stringutils.CapitalizeFirst(stringutils.HumanizeSnakeCase(stringOf(result["reportKey"])))
	}
	if title == "" {
		title = "Report run"
	}

	return &assistantartifact.Artifact{
		Kind:             assistantartifact.KindReportRun,
		Status:           runStatus(boolOf(result["finished"]), stringOf(result["status"])),
		Title:            artifactTitle(title),
		Payload:          payload,
		SourceToolCallID: callID,
	}
}

// runStatus follows the report run: Pending until it finishes, Failed when
// it did not succeed, Ready when there is something to download.
func runStatus(finished bool, status string) assistantartifact.Status {
	if !finished {
		return assistantartifact.StatusPending
	}
	switch strings.ToLower(status) {
	case "failed", "canceled", "cancelled", "expired":
		return assistantartifact.StatusFailed
	default:
		return assistantartifact.StatusReady
	}
}

// entityCardArtifact shows one record a get tool fetched. A result without
// an id is a view rather than a record (a board, a schedule) and is left to
// the transcript.
func entityCardArtifact(callID, toolName string, result map[string]any) *assistantartifact.Artifact {
	if stringOf(result["id"]) == "" {
		return nil
	}

	entity := strings.TrimPrefix(toolName, getToolPrefix)
	payload := map[string]any{
		"entity": entity,
		"record": result,
	}
	if payloadSize(payload) > assistantartifact.MaxPayloadBytes {
		return nil
	}

	return &assistantartifact.Artifact{
		Kind:   assistantartifact.KindEntityCard,
		Status: assistantartifact.StatusReady,
		Title: artifactTitle(
			stringutils.CapitalizeFirst(stringutils.HumanizeSnakeCase(entity)) + " " + recordLabel(result),
		),
		Payload:          payload,
		SourceToolCallID: callID,
	}
}

// draftArtifact views an outbound message proposal as a draft. Only the
// tools in draftSpecs produce one; a dispatch or billing write is decided
// from its card.
func draftArtifact(proposal *agent.AgentProposal) *assistantartifact.Artifact {
	if proposal == nil {
		return nil
	}
	spec, ok := draftSpecs[proposal.ToolName]
	if !ok {
		return nil
	}

	payload := map[string]any{
		"tool":      proposal.ToolName,
		"arguments": proposal.ToolParams,
		"rationale": proposal.Rationale,
	}
	subject := ""
	if spec.subjectKey != "" {
		subject = stringOf(proposal.ToolParams[spec.subjectKey])
		payload["subject"] = subject
	}
	if spec.bodyKey != "" {
		payload["body"] = stringOf(proposal.ToolParams[spec.bodyKey])
	}
	if spec.toKey != "" {
		payload["to"] = proposal.ToolParams[spec.toKey]
	}

	title := spec.title
	if subject != "" {
		title = subject
	}

	artifact := &assistantartifact.Artifact{
		Kind:       assistantartifact.KindEmailDraft,
		Status:     draftStatus(proposal.Status),
		Title:      artifactTitle(title),
		Payload:    payload,
		ProposalID: proposal.ID,
		RunID:      proposal.RunID,
		MessageID:  proposal.SourceMessageID,
	}
	if proposal.PlanID != nil {
		artifact.PlanID = *proposal.PlanID
	}

	return artifact
}

func planArtifact(plan *agent.AgentPlan, proposals []*agent.AgentProposal) *assistantartifact.Artifact {
	steps := make([]map[string]any, 0, len(proposals))
	for _, proposal := range proposals {
		if proposal == nil || proposal.PlanID == nil || *proposal.PlanID != plan.ID {
			continue
		}
		steps = append(steps, map[string]any{
			"step":       proposal.PlanStep,
			"proposalId": proposal.ID.String(),
			"toolName":   proposal.ToolName,
			"rationale":  proposal.Rationale,
		})
	}

	return &assistantartifact.Artifact{
		Kind:   assistantartifact.KindPlan,
		Status: planStatus(plan.Status),
		Title:  artifactTitle(plan.Title),
		Payload: map[string]any{
			"title":     plan.Title,
			"summary":   plan.Summary,
			"stepCount": plan.StepCount,
			"steps":     steps,
		},
		PlanID: plan.ID,
		RunID:  plan.RunID,
	}
}

// draftStatus is where a draft is, read from the proposal that carries the
// decision. A draft never holds its own decision state.
func draftStatus(status agent.ProposalStatus) assistantartifact.Status {
	switch status {
	case agent.ProposalStatusExecuted, agent.ProposalStatusSimulated:
		return assistantartifact.StatusSent
	case agent.ProposalStatusRejected, agent.ProposalStatusExpired,
		agent.ProposalStatusSuperseded, agent.ProposalStatusExecutionFailed,
		agent.ProposalStatusSkipped:
		return assistantartifact.StatusFailed
	case agent.ProposalStatusPending, agent.ProposalStatusAccepted, agent.ProposalStatusModified:
		return assistantartifact.StatusPending
	default:
		return assistantartifact.StatusPending
	}
}

func planStatus(status agent.PlanStatus) assistantartifact.Status {
	switch status {
	case agent.PlanStatusCompleted:
		return assistantartifact.StatusReady
	case agent.PlanStatusFailed, agent.PlanStatusRejected, agent.PlanStatusExpired:
		return assistantartifact.StatusFailed
	case agent.PlanStatusPending, agent.PlanStatusApproved:
		return assistantartifact.StatusPending
	default:
		return assistantartifact.StatusPending
	}
}

// fitRows halves the rows under key until the payload is within bound, and
// says so. A preview the tool already capped rarely needs it; a wide one
// with long text cells can.
func fitRows(payload map[string]any, key string) {
	rows, _ := payload[key].([]any)
	for payloadSize(payload) > assistantartifact.MaxPayloadBytes && len(rows) > minPreviewRows {
		rows = rows[:len(rows)/2]
		payload[key] = rows
		payload["truncated"] = true
	}
	if len(rows) <= minPreviewRows && payloadSize(payload) > assistantartifact.MaxPayloadBytes {
		payload[key] = []any{}
		payload["truncated"] = true
	}
}

func payloadSize(payload map[string]any) int {
	encoded, err := sonic.Marshal(payload)
	if err != nil {
		return assistantartifact.MaxPayloadBytes + 1
	}

	return len(encoded)
}

// toJSONMap reads a tool result in its JSON form, which is the contract the
// tool publishes; its Go type is the tool's own business.
func toJSONMap(data any) (map[string]any, bool) {
	if m, ok := data.(map[string]any); ok {
		return m, true
	}

	encoded, err := sonic.Marshal(data)
	if err != nil {
		return nil, false
	}
	var out map[string]any
	if err = sonic.Unmarshal(encoded, &out); err != nil || out == nil {
		return nil, false
	}

	return out, true
}

func recordLabel(record map[string]any) string {
	for _, key := range recordLabelKeys {
		if value := stringOf(record[key]); value != "" {
			return value
		}
	}
	if first, last := stringOf(record["firstName"]), stringOf(record["lastName"]); first != "" || last != "" {
		return strings.TrimSpace(first + " " + last)
	}

	return stringOf(record["id"])
}

func artifactTitle(title string) string {
	title = strings.TrimSpace(title)
	if title == "" {
		title = "Untitled"
	}

	return stringutils.TruncateRunes(title, maxArtifactTitleRunes)
}

func stringOf(value any) string {
	s, _ := value.(string)

	return strings.TrimSpace(s)
}

func boolOf(value any) bool {
	b, _ := value.(bool)

	return b
}

func toAssistantArtifact(artifact *assistantartifact.Artifact) services.AssistantArtifact {
	return services.AssistantArtifact{
		ID:               artifact.ID,
		ThreadID:         artifact.ThreadID,
		MessageID:        artifact.MessageID,
		RunID:            artifact.RunID,
		ProposalID:       artifact.ProposalID,
		PlanID:           artifact.PlanID,
		Kind:             artifact.Kind,
		Status:           artifact.Status,
		Title:            artifact.Title,
		Payload:          artifact.Payload,
		SourceToolCallID: artifact.SourceToolCallID,
		Pinned:           artifact.Pinned,
		CreatedAt:        artifact.CreatedAt,
		UpdatedAt:        artifact.UpdatedAt,
	}
}

// ListThreadArtifacts reads a thread's artifacts with each draft's and
// plan's status read from the proposal or plan it views, so the pane shows
// a draft as sent the moment the decision ran, without a second write.
func (s *Service) ListThreadArtifacts(
	ctx context.Context,
	req repositories.GetThreadRequest,
) ([]services.AssistantArtifact, error) {
	if _, err := s.conversations.GetThread(ctx, req); err != nil {
		return nil, err
	}
	if s.artifacts == nil {
		return []services.AssistantArtifact{}, nil
	}

	artifacts, err := s.artifacts.ListByThread(ctx, repositories.ListArtifactsRequest{
		ThreadID:   req.ID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	s.followDecisions(ctx, req, artifacts)

	out := make([]services.AssistantArtifact, 0, len(artifacts))
	for _, artifact := range artifacts {
		out = append(out, toAssistantArtifact(artifact))
	}

	return out, nil
}

// followDecisions overlays each draft's and plan's status from the record
// that carries its decision. A read that fails leaves the stored status,
// which is what the pane showed last time.
func (s *Service) followDecisions(
	ctx context.Context,
	req repositories.GetThreadRequest,
	artifacts []*assistantartifact.Artifact,
) {
	var wantProposals, wantPlans bool
	for _, artifact := range artifacts {
		wantProposals = wantProposals || !artifact.ProposalID.IsNil()
		wantPlans = wantPlans || (artifact.Kind == assistantartifact.KindPlan && !artifact.PlanID.IsNil())
	}

	if wantProposals && s.proposals != nil {
		proposals, err := s.proposals.ListByThread(ctx, repositories.ListAgentProposalsByThreadRequest{
			ThreadID:   req.ID,
			TenantInfo: req.TenantInfo,
		})
		if err != nil {
			s.logger.Warn("artifact statuses could not follow proposals", zap.Error(err))
		} else {
			byID := make(map[pulid.ID]agent.ProposalStatus, len(proposals))
			for _, proposal := range proposals {
				byID[proposal.ID] = proposal.Status
			}
			for _, artifact := range artifacts {
				if status, ok := byID[artifact.ProposalID]; ok && artifact.Kind == assistantartifact.KindEmailDraft {
					artifact.Status = draftStatus(status)
				}
			}
		}
	}

	if wantPlans && s.plans != nil {
		plans, err := s.plans.ListByThread(ctx, repositories.ListAgentPlansByThreadRequest{
			ThreadID:   req.ID,
			TenantInfo: req.TenantInfo,
		})
		if err != nil {
			s.logger.Warn("artifact statuses could not follow plans", zap.Error(err))
		} else {
			byID := make(map[pulid.ID]agent.PlanStatus, len(plans))
			for _, plan := range plans {
				byID[plan.ID] = plan.Status
			}
			for _, artifact := range artifacts {
				if status, ok := byID[artifact.PlanID]; ok && artifact.Kind == assistantartifact.KindPlan {
					artifact.Status = planStatus(status)
				}
			}
		}
	}
}

func (s *Service) PinArtifact(
	ctx context.Context,
	req repositories.GetThreadRequest,
	artifactID pulid.ID,
	pinned bool,
) (*services.AssistantArtifact, error) {
	if _, err := s.conversations.GetThread(ctx, req); err != nil {
		return nil, err
	}
	if s.artifacts == nil {
		return nil, errortypes.NewNotFoundError("Artifact not found")
	}

	artifact, err := s.artifacts.SetPinned(ctx, repositories.SetArtifactPinnedRequest{
		ID:         artifactID,
		ThreadID:   req.ID,
		TenantInfo: req.TenantInfo,
		Pinned:     pinned,
	})
	if err != nil {
		return nil, err
	}

	out := toAssistantArtifact(artifact)

	return &out, nil
}
