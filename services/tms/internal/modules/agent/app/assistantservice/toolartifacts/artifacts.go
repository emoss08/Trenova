package toolartifacts

import (
	"cmp"
	"errors"
	"fmt"
	"maps"
	"math"
	"slices"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/replycheck"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/artifactpayload"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/recordviews"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentplan"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/domain/codemode"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/pagedraft"
	"github.com/emoss08/trenova/pkg/productguide"
	"github.com/emoss08/trenova/shared/inflect"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/textcase"
	"github.com/emoss08/trenova/shared/typeutils"
)

// The tools whose results a person would rather see than read. A preview is
// a table; a run is a card with a download; a get is the record itself. The
// mapper reads each result in its JSON form, exactly as the model does, so
// it depends on the tool's contract rather than on its Go types.
const (
	toolPreviewReport  = "report_preview"
	ToolRunReport      = "report_run"
	toolGetReportRun   = "report_get_run"
	getToolPrefix      = "get_"
	listToolPrefix     = "list_"
	searchToolPrefix   = "search_"
	toolComposeView    = "report_compose_table_view"
	toolExplainRate    = "rate_explain"
	toolCompareRuns    = "report_compare_runs"
	toolOpenPage       = "agent_open_page"
	toolRankCandidates = "dispatch_rank_candidates"

	paramBody    = "body"
	paramSubject = "subject"

	minPreviewRows = 1

	payloadProposalID  = "proposalId"
	payloadProposalIDs = "proposalIds"
	payloadPlanID      = "planId"
	payloadToolName    = "toolName"
	payloadRationale   = "rationale"

	// billingQueueRecordEntity is the record a billing queue table's rows are.
	billingQueueRecordEntity = "billing_queue_item"

	// payloadRanked marks a table whose order is the answer: drivers ranked
	// for a move, best first.
	payloadRanked = "ranked"
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
	"customer_send_email": {
		title:      "Customer update",
		subjectKey: paramSubject,
		bodyKey:    paramBody,
	},
	"detention_send_notice": {title: "Detention notice"},
	// The reply's subject and recipient come from the message it answers,
	// not from the proposal, so the draft carries the body alone.
	"inbox_send_reply": {title: "Inbox reply", bodyKey: paramBody},
	"shipment_request_documents": {
		title:      "Document request",
		subjectKey: paramSubject,
		bodyKey:    paramBody,
		toKey:      "to",
	},
}

// LinkedArtifacts is every artifact the turn's replies point to. Only the
// conversation's own agent answers the person: an agent it handed a task to
// reports back to that agent, and the records it cited while doing so are its
// working, not the answer. They stay only if the reply itself uses them.
func LinkedArtifacts(messages []conversation.Message) map[string]bool {
	linked := map[string]bool{}
	for idx := range messages {
		if messages[idx].Role != conversation.RoleAssistant || messages[idx].DelegateCallID != "" {
			continue
		}
		for _, id := range replycheck.ArtifactRefIDs(messages[idx].Content) {
			linked[id] = true
		}
	}

	return linked
}

// SameView says a new version would show exactly what the last one shows:
// the same title over the same data, so there is nothing new to keep.
func SameView(previous, next *assistantartifact.Artifact) bool {
	if previous == nil || previous.Kind != next.Kind || previous.Title != next.Title {
		return false
	}
	// ConfigStd sorts map keys; the fast path writes them in whatever order
	// it meets them, and two identical views would read as different.
	before, err := sonic.ConfigStd.Marshal(previous.Payload)
	if err != nil {
		return false
	}
	after, err := sonic.ConfigStd.Marshal(next.Payload)
	if err != nil {
		return false
	}

	return string(before) == string(after)
}

// ShownArtifact is how the model is told what the person now sees.
func ShownArtifact(artifact *assistantartifact.Artifact) *agentruntimeports.ShownArtifact {
	return &agentruntimeports.ShownArtifact{
		ID:           artifact.ID,
		Kind:         string(artifact.Kind),
		Title:        artifact.Title,
		Labels:       tableLabels(artifact),
		Rows:         int(recordviews.NumberOf(artifact.Payload, "rowCount")),
		Actionable:   actionableTable(artifact),
		Opens:        opensView(artifact),
		OpensRecords: opensRecords(artifact),
		Ranked:       rankedTable(artifact),
		Paged:        artifactpayload.PagedTable(artifact),
		Written:      writtenPhrase(artifact),
		Lineage:      lineageRoot(artifact),
	}
}

func lineageRoot(artifact *assistantartifact.Artifact) pulid.ID {
	if artifact.LineageID.IsNil() {
		return artifact.ID
	}

	return artifact.LineageID
}

// MustBePointedTo is a lookup the model was told to point to rather than
// reprint (shownNote's opens, actionable, records, ranked and long-table branches),
// which stays beside the conversation whether or not the reply managed the
// link.
func MustBePointedTo(artifact *assistantartifact.Artifact) bool {
	return ShownArtifact(artifact).MustBeLinked()
}

func tabularKind(kind assistantartifact.Kind) bool {
	return kind == assistantartifact.KindTableView || kind == assistantartifact.KindReportPreview
}

// rankedTable is a table whose order is what the person decides from: the
// drivers ranked for a move. Five rows are few enough to reprint, and the
// Desk did, as a markdown table that kept the order and lost the findings
// behind it and the link to each driver.
func rankedTable(artifact *assistantartifact.Artifact) bool {
	ranked, _ := artifact.Payload[payloadRanked].(bool)

	return artifact.Kind == assistantartifact.KindTableView && ranked
}

// opensView is a composed view: a table that opens the live page.
func opensView(artifact *assistantartifact.Artifact) bool {
	_, opens := artifact.Payload["path"]

	return artifact.Kind == assistantartifact.KindTableView && opens
}

// actionableTable is a table whose rows the person acts on where it opens:
// the billing queue, where items are selected, reviewed and approved. The
// design draws it with a checkbox column and a bulk bar, which a markdown
// table in the reply could never offer.
func actionableTable(artifact *assistantartifact.Artifact) bool {
	if artifact.Kind != assistantartifact.KindTableView {
		return false
	}
	// A composed view opens the live table; its rows are a preview of that,
	// never the answer to reprint.
	if _, opens := artifact.Payload["path"]; opens {
		return true
	}

	return typeutils.StringOfTrimmed(artifact.Payload["recordEntity"]) == billingQueueRecordEntity
}

// opensRecords is a table whose rows open beside the conversation as the
// record itself, drawn by the Desk's card for that kind of record: a formula
// template opens as its formula, its variables and a price for a sample load,
// which a markdown table in the reply could never hold. Which kinds have a
// card is read from the Desk's card registry through the product guide.
func opensRecords(artifact *assistantartifact.Artifact) bool {
	if artifact.Kind != assistantartifact.KindTableView {
		return false
	}
	if _, opens := artifact.Payload["path"]; opens {
		return false
	}
	link, ok := productguide.Default.Record(
		typeutils.StringOfTrimmed(artifact.Payload["recordEntity"]),
	)

	return ok && link.DeskCard
}

func RequestedArtifact(
	callID string,
	request agentruntimeports.DecisionRequest,
	stored []*agent.AgentProposal,
) (*assistantartifact.Artifact, error) {
	byID := make(map[pulid.ID]*agent.AgentProposal, len(stored))
	for _, candidate := range stored {
		if candidate != nil {
			byID[candidate.ID] = candidate
		}
	}

	if request.PlanID.IsNotNil() {
		return planDecisionArtifact(callID, request.PlanID, stored)
	}
	if len(request.ProposalIDs) > 1 {
		return bunchDecisionArtifact(callID, request.ProposalIDs, byID)
	}

	proposal, err := pendingByID(byID, request.ProposalID)
	if err != nil {
		return nil, err
	}

	return decisionRequestArtifact(callID, proposal), nil
}

func pendingByID(
	byID map[pulid.ID]*agent.AgentProposal,
	id pulid.ID,
) (*agent.AgentProposal, error) {
	proposal, ok := byID[id]
	switch {
	case !ok:
		return nil, ErrUnknownProposal
	case proposal.Status != agent.ProposalStatusPending:
		return nil, fmt.Errorf("%w: it is %s", ErrProposalDecided,
			strings.ToLower(string(proposal.Status)))
	}

	return proposal, nil
}

func planDecisionArtifact(
	callID string,
	planID pulid.ID,
	stored []*agent.AgentProposal,
) (*assistantartifact.Artifact, error) {
	steps := make([]*agent.AgentProposal, 0, len(stored))
	for _, proposal := range stored {
		if proposal != nil && proposal.PlanID != nil && *proposal.PlanID == planID &&
			proposal.Status == agent.ProposalStatusPending {
			steps = append(steps, proposal)
		}
	}
	if len(steps) == 0 {
		return nil, ErrUnknownPlan
	}
	slices.SortStableFunc(steps, func(a, b *agent.AgentProposal) int {
		return cmp.Compare(a.PlanStep, b.PlanStep)
	})

	artifact := decisionRequestArtifact(callID, steps[0])
	artifact.PlanID = planID
	artifact.Title = assistantartifact.NormalizeTitle(
		fmt.Sprintf("Plan: %s", countChanges(len(steps))),
	)
	artifact.Payload[payloadPlanID] = planID.String()
	artifact.Payload[payloadProposalIDs] = proposalIDStrings(steps)
	artifact.Payload[payloadToolName] = agentplan.PlanToolName

	return artifact, nil
}

func bunchDecisionArtifact(
	callID string,
	ids []pulid.ID,
	byID map[pulid.ID]*agent.AgentProposal,
) (*assistantartifact.Artifact, error) {
	proposals := make([]*agent.AgentProposal, 0, len(ids))
	for _, id := range ids {
		proposal, err := pendingByID(byID, id)
		if err != nil {
			return nil, err
		}
		switch {
		case proposal.PlanID != nil:
			return nil, errProposalInPlan
		case len(proposals) > 0 && proposal.ToolName != proposals[0].ToolName:
			return nil, ErrMixedTools
		}
		proposals = append(proposals, proposal)
	}

	artifact := decisionRequestArtifact(callID, proposals[0])
	artifact.Title = assistantartifact.NormalizeTitle(fmt.Sprintf(
		"%s (%d)",
		assistantartifact.PhraseTitle(
			agenttoolports.ReadingOrder(proposals[0].ToolName),
		),
		len(proposals),
	))
	artifact.Payload[payloadProposalIDs] = proposalIDStrings(proposals)

	return artifact, nil
}

func proposalIDStrings(proposals []*agent.AgentProposal) []string {
	ids := make([]string, 0, len(proposals))
	for _, proposal := range proposals {
		ids = append(ids, proposal.ID.String())
	}

	return ids
}

func countChanges(n int) string {
	if n == 1 {
		return "1 change"
	}

	return fmt.Sprintf("%d changes", n)
}

func decisionRequestArtifact(
	callID string,
	proposal *agent.AgentProposal,
) *assistantartifact.Artifact {
	artifact := &assistantartifact.Artifact{
		Kind:   assistantartifact.KindDecisionRequest,
		Status: artifactpayload.DecisionStatus(proposal.Status),
		Title:  assistantartifact.PhraseTitle(agenttoolports.ReadingOrder(proposal.ToolName)),
		Payload: map[string]any{
			payloadProposalID: proposal.ID.String(),
			payloadToolName:   proposal.ToolName,
			payloadRationale:  proposal.Rationale,
		},
		ProposalID:       proposal.ID,
		RunID:            proposal.RunID,
		SourceToolCallID: callID,
	}
	if proposal.PlanID != nil {
		artifact.PlanID = *proposal.PlanID
	}

	return artifact
}

var (
	ErrDecisionUnavailable = errors.New("its proposal could not be read")
	ErrUnknownProposal     = errors.New("there is no proposal with that id in this conversation")
	ErrProposalDecided     = errors.New("that proposal is no longer waiting on the person")
	ErrDecisionNotShown    = errors.New("its card could not be saved")
	ErrUnknownPlan         = errors.New(
		"there is no plan with that id waiting in this conversation",
	)
	errProposalInPlan = errors.New("one of them is a step of a plan; ask with its planId")
	ErrMixedTools     = errors.New("proposals of different tools cannot share one card")
)

var (
	ErrUnknownDocument error = &agentruntimeports.PublishMistake{
		Reason: "there is no document with that artifactId in this conversation; leave " +
			"artifactId out to publish a new one",
	}
	ErrDocumentNotKept       = errors.New("it could not be saved")
	ErrUnknownAnalysis error = &agentruntimeports.PublishMistake{
		Reason: "an analysis it names is not one an analysis_run result kept in this " +
			"conversation; name each by the artifactId of its analysis",
	}
)

// DocumentArtifact is a write-up the model published, kept as markdown with
// what the Desk sets around it: its type, who wrote it and from what, and the
// sources its citation marks point to.
func DocumentArtifact(
	callID string,
	document *agentruntimeports.PublishedDocument,
	author string,
) *assistantartifact.Artifact {
	sources := make([]assistantartifact.DocumentSource, 0, len(document.Sources))
	for _, source := range document.Sources {
		shown := assistantartifact.DocumentSource{
			N:      source.N,
			Tool:   source.Tool,
			Label:  source.Label,
			Detail: source.Detail,
		}
		if !source.ArtifactID.IsNil() {
			shown.ArtifactID = source.ArtifactID.String()
		}
		sources = append(sources, shown)
	}
	slices.SortFunc(sources, func(a, b assistantartifact.DocumentSource) int {
		return cmp.Compare(a.N, b.N)
	})
	docType := strings.TrimSpace(document.DocType)
	if docType == "" {
		docType = "Document"
	}

	return &assistantartifact.Artifact{
		Kind:   assistantartifact.KindDocument,
		Status: assistantartifact.StatusReady,
		Title:  assistantartifact.NormalizeTitle(document.Title),
		Payload: map[string]any{
			assistantartifact.DocumentFormat:      "markdown",
			assistantartifact.DocumentBody:        document.Body,
			assistantartifact.DocumentType:        docType,
			assistantartifact.DocumentAuthor:      author,
			assistantartifact.DocumentBasis:       strings.TrimSpace(document.Basis),
			assistantartifact.DocumentSources:     sources,
			assistantartifact.DocumentCitations:   assistantartifact.CitedNumbers(document.Body),
			assistantartifact.DocumentEditedBy:    assistantartifact.DocumentEditedByAgent,
			assistantartifact.DocumentEditor:      author,
			assistantartifact.DocumentVersionNote: "First draft",
		},
		SourceToolCallID: callID,
	}
}

// ArtifactFromObservation turns a finished query tool into what the pane
// shows for it, or nil when the result is one to read rather than see.
func ArtifactFromObservation(
	observation agentruntimeports.ToolObservation,
) *assistantartifact.Artifact {
	if observation.Failed || observation.Data == nil || observation.Call.ID == "" {
		return nil
	}

	document, ok := artifactpayload.ToJSONDocument(observation.Data)
	if !ok {
		return nil
	}
	result := document.Fields

	name := observation.Call.Name
	switch {
	case name == toolPreviewReport:
		return previewArtifact(observation.Call.ID, result, observation.Call.Arguments)
	case name == toolRankCandidates:
		return candidateRankingArtifact(observation.Call.ID, result)
	case name == ToolRunReport || name == toolGetReportRun:
		return RunArtifact(observation.Call.ID, result)
	case strings.HasPrefix(agenttoolports.ReadingOrder(name), getToolPrefix) && isRowSet(result):
		return tableArtifact(observation.Call.ID, name, result)
	case strings.HasPrefix(agenttoolports.ReadingOrder(name), getToolPrefix):
		return EntityCardArtifact(observation.Call.ID, name, document)
	case name == toolExplainRate:
		return rateArtifact(observation.Call.ID, result)
	case name == toolCompareRuns:
		return runDiffArtifact(observation.Call.ID, result)
	case name == toolOpenPage:
		return navigationArtifact(observation.Call.ID, result)
	case name == toolComposeView:
		return composedViewArtifact(observation.Call.ID, result)
	case pagedraft.IsEditTool(name):
		return draftEditArtifact(observation.Call.ID, result)
	case name == toolDraftEDIConnection:
		return connectionDraftArtifact(observation.Call.ID, result)
	case name == codemode.ToolRunScript:
		return analysisArtifact(observation.Call.ID, result, observation.Call.Arguments)
	case name == agentextension.ToolReadPortal:
		return portalTaskArtifact(observation.Call.ID, result)
	case strings.HasPrefix(agenttoolports.ReadingOrder(name), listToolPrefix),
		strings.HasPrefix(agenttoolports.ReadingOrder(name), searchToolPrefix):
		if card := soleRecordCard(observation.Call.ID, name, result); card != nil {
			return card
		}
		return tableArtifact(observation.Call.ID, name, result)
	default:
		return nil
	}
}

func isRowSet(result map[string]any) bool {
	_, rows := result["items"].([]any)

	return rows && len(artifactpayload.StringsOf(result["columns"])) > 0
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
//
// The rows are the projection a person reads, not the ones the model did:
// see artifact_display.go. A table with nothing readable left in it is not
// one either.
func tableArtifact(callID, toolName string, result map[string]any) *assistantartifact.Artifact {
	rows, ok := result["items"].([]any)
	if !ok || len(rows) == 0 {
		return nil
	}
	declared := artifactpayload.StringsOf(result["columns"])
	if len(declared) == 0 {
		return nil
	}

	entity := listedEntity(toolName)
	projection := artifactpayload.ProjectTable(entity, declared, rows)
	if len(projection.Columns) == 0 {
		return nil
	}

	payload := map[string]any{
		"display":     assistantartifact.DisplayVersion,
		"tool":        toolName,
		"entity":      entity,
		"columns":     projection.Columns,
		"rows":        projection.Rows,
		"rowCount":    result["count"],
		"searchedFor": artifactpayload.StringsOf(result["searchedFor"]),
	}
	if projection.RecordEntity != "" {
		payload["recordEntity"] = projection.RecordEntity
	}
	fitRows(payload, "rows")
	artifactpayload.NotePaging(payload, result)

	return &assistantartifact.Artifact{
		Kind:             assistantartifact.KindTableView,
		Status:           assistantartifact.StatusReady,
		Title:            assistantartifact.RecordsTitle(entity),
		Payload:          payload,
		SourceToolCallID: callID,
	}
}

// listedEntity is what a list, search or get tool names its records by:
// "formula_templates" for formula_list_templates, and "invoices" for
// invoice_get, whose name is singular because it reads one invoice or many.
func listedEntity(toolName string) string {
	reading := agenttoolports.ReadingOrder(toolName)
	if entity, read := strings.CutPrefix(reading, getToolPrefix); read {
		return inflect.PluralNoun(inflect.SingularNoun(entity))
	}

	return strings.TrimPrefix(strings.TrimPrefix(reading, listToolPrefix), searchToolPrefix)
}

// soleRecordCard is a list or search that found exactly one record of a kind
// the Desk draws as a card, drawn as that card rather than as a table of one
// row. Asked to show a formula template by name, the Desk found it with
// formula_list_templates and kept a one-row table, and the person had to open
// the row to see the template they had asked for. A page of one that has more
// behind it stays a table: it is not the only record, only the first.
func soleRecordCard(callID, toolName string, result map[string]any) *assistantartifact.Artifact {
	rows, ok := result["items"].([]any)
	if !ok || len(rows) != 1 || result["hasMore"] == true {
		return nil
	}
	if _, counted := result["count"]; counted && recordviews.NumberOf(result, "count") != 1 {
		return nil
	}
	record, ok := rows[0].(map[string]any)
	if !ok {
		return nil
	}
	entity := artifactpayload.RecordEntityOf(listedEntity(toolName))
	if link, carded := productguide.Default.Record(entity); !carded || !link.DeskCard {
		return nil
	}

	return RecordCardArtifact(&RecordCard{
		CallID:   callID,
		ToolName: toolName,
		Entity:   entity,
		Record:   record,
		Order:    declaredOrder(artifactpayload.StringsOf(result["columns"]), record),
	})
}

// declaredOrder is a row's fields in the order its tool declared its
// columns, then any it did not declare, alphabetically.
func declaredOrder(declared []string, record map[string]any) []string {
	order := make([]string, 0, len(record))
	for _, key := range declared {
		if _, ok := record[key]; ok && !slices.Contains(order, key) {
			order = append(order, key)
		}
	}
	for _, key := range slices.Sorted(maps.Keys(record)) {
		if !slices.Contains(order, key) {
			order = append(order, key)
		}
	}

	return order
}

// candidateColumns are a ranked driver as a dispatcher weighs one: where it
// stands, who it is, the console's verdict and score, and the figures the
// score was made from. The tractor and trailer are ids only, which a person
// cannot read, so they stay in the model's copy for dispatch_assign_move.
var candidateColumns = []string{
	"rank",
	"driver",
	"fit",
	"score",
	"deadheadMiles",
	"minutesOfSlack",
	"driveRemainingHours",
	"projectedArrival",
	"finding",
}

// candidateRankingArtifact is a move's ranked drivers as a table, best
// first.
//
// It used to be nothing at all. Asked to rank drivers for a load, the Desk
// read the ranking and wrote it out again as a markdown table: the order
// survived, the findings behind each verdict and the link to each driver did
// not, and the dispatcher had nothing beside the conversation to pick from.
// The rank is a column of its own because the order is the answer, and a
// table re-sorted by deadhead would otherwise lose it.
func candidateRankingArtifact(callID string, result map[string]any) *assistantartifact.Artifact {
	candidates, ok := result["candidates"].([]any)
	if !ok || len(candidates) == 0 {
		return nil
	}

	rows := make([]any, 0, len(candidates))
	for _, raw := range candidates {
		candidate, isRecord := raw.(map[string]any)
		if !isRecord {
			continue
		}
		row := map[string]any{
			artifactpayload.RecordIDKey: candidate["workerId"],
			"rank":                      float64(len(rows) + 1),
			"driver":                    candidate["workerName"],
			"fit":                       candidate["verdict"],
			"score":                     candidate["score"],
			"deadheadMiles":             candidate["deadheadMiles"],
			"minutesOfSlack":            candidate["minutesOfSlack"],
			"driveRemainingHours": math.Round(
				recordviews.NumberOf(candidate, "driveRemainingHours")*10,
			) / 10,
			"finding": leadingFinding(candidate),
		}
		// An unknown arrival is a phrase in the model's copy; in a column of
		// times it would be the one cell that is not one.
		if arrival := recordviews.NumberOf(candidate, "projectedArrival"); arrival > 0 {
			row["projectedArrival"] = arrival
		}
		rows = append(rows, row)
	}

	projection := artifactpayload.ProjectTable("workers", candidateColumns, rows)
	if len(projection.Columns) == 0 {
		return nil
	}

	payload := map[string]any{
		"display":     assistantartifact.DisplayVersion,
		"tool":        toolRankCandidates,
		"entity":      "workers",
		"columns":     projection.Columns,
		"rows":        projection.Rows,
		"rowCount":    float64(len(projection.Rows)),
		"searchedFor": []string{},
		payloadRanked: true,
	}
	if projection.RecordEntity != "" {
		payload["recordEntity"] = projection.RecordEntity
	}
	fitRows(payload, "rows")
	artifactpayload.NotePaging(payload, result)

	return &assistantartifact.Artifact{
		Kind:             assistantartifact.KindTableView,
		Status:           assistantartifact.StatusReady,
		Title:            assistantartifact.NormalizeTitle("Drivers ranked for the move"),
		Payload:          payload,
		SourceToolCallID: callID,
	}
}

// leadingFinding is the one finding a dispatcher reads first: what rules the
// driver out, or failing that what to watch. A note that changes nothing is
// left in the model's copy.
func leadingFinding(candidate map[string]any) string {
	findings := recordviews.ListOf(candidate["findings"])
	for _, severity := range []string{"Block", "Warn"} {
		for _, finding := range findings {
			if recordviews.TextOf(finding, "severity") == severity {
				return recordviews.TextOf(finding, "message")
			}
		}
	}

	return ""
}

// composedViewArtifact is a described view as something to open.
//
// The pane shows what it was narrowed to and what could not be, how many rows
// it holds and the first few, with the link to the live table. It is a
// table_view like a list result, because to the reader it is the same thing
// arrived at a different way — except that this one opens rather than being a
// snapshot, which its path says.
func composedViewArtifact(callID string, result map[string]any) *assistantartifact.Artifact {
	path := typeutils.StringOfTrimmed(result["path"])
	if !assistantartifact.IsAppPath(path) {
		return nil
	}

	entity := typeutils.StringOfTrimmed(result["entity"])
	payload := map[string]any{
		"display":     assistantartifact.DisplayVersion,
		"entity":      entity,
		"path":        path,
		"explanation": typeutils.StringOfTrimmed(result["explanation"]),
		"terms":       artifactpayload.StringsOf(result["terms"]),
		"filterCount": result["filterCount"],
		"unresolved":  result["unresolved"],
	}
	if count, counted := result["count"]; counted {
		payload["rowCount"] = count
		payload["countCapped"] = typeutils.BoolOf(result["countCapped"])
	}
	if rows, ok := result["items"].([]any); ok && len(rows) > 0 {
		projection := artifactpayload.ProjectTable(
			entity,
			artifactpayload.StringsOf(result["columns"]),
			rows,
		)
		if len(projection.Columns) > 0 {
			payload["columns"] = projection.Columns
			payload["rows"] = projection.Rows
			if projection.RecordEntity != "" {
				payload["recordEntity"] = projection.RecordEntity
			}
			fitRows(payload, "rows")
		}
	}

	title := assistantartifact.RecordsTitle(entity)
	if explanation := typeutils.StringOfTrimmed(result["explanation"]); explanation != "" {
		title = explanation
	}

	return &assistantartifact.Artifact{
		Kind:             assistantartifact.KindTableView,
		Status:           assistantartifact.StatusReady,
		Title:            assistantartifact.NormalizeTitle(title),
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
	// Nothing to explain is a sentence, not a ledger of zeroes. A shipment
	// priced from its formula template has a note too, and a ledger.
	pricedBy := recordviews.ObjectOf(result["pricedBy"])
	if typeutils.StringOfTrimmed(result["note"]) != "" && pricedBy == nil {
		return nil
	}
	components, _ := result["components"].([]any)
	if len(components) == 0 {
		return nil
	}

	payload := map[string]any{
		"shipmentId": typeutils.StringOfTrimmed(result["shipmentId"]),
		"side":       typeutils.StringOfTrimmed(result["side"]),
		"currency":   typeutils.StringOfTrimmed(result["currency"]),
		"winner":     result["winner"],
		"tieBreak":   typeutils.StringOfTrimmed(result["tieBreak"]),
		"rejected":   result["rejected"],
		"components": components,
		"guardrails": result["guardrails"],
		"totals":     result["totals"],
		"warnings":   artifactpayload.StringsOf(result["warnings"]),
	}
	if pricedBy != nil {
		payload["pricedBy"] = pricedBy
	}
	fitRows(payload, "components")

	return &assistantartifact.Artifact{
		Kind:             assistantartifact.KindRateExplanation,
		Status:           assistantartifact.StatusReady,
		Title:            assistantartifact.NormalizeTitle("Rate breakdown"),
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
		"keys":      artifactpayload.StringsOf(result["keys"]),
		"measures":  artifactpayload.StringsOf(result["measures"]),
		"summary":   result["summary"],
		"changes":   changes,
		"totals":    totals,
		"truncated": result["truncated"],
		"note":      typeutils.StringOfTrimmed(result["note"]),
	}
	// The changes are already sorted by risk and then by size, so halving the
	// list drops the smallest movements rather than an arbitrary tail.
	fitRows(payload, "changes")

	return &assistantartifact.Artifact{
		Kind:             assistantartifact.KindRunDiff,
		Status:           assistantartifact.StatusReady,
		Title:            assistantartifact.NormalizeTitle(runDiffTitle(result)),
		Payload:          payload,
		SourceToolCallID: callID,
	}
}

// navigationArtifact is where an agent took the person.
//
// agent_open_page has already checked the page exists and that the person may open
// it; the artifact is how the app hears about it. The live one moves the app
// there once, and the one read back from history is a card with a link, so
// reopening a conversation never drags anybody anywhere.
func navigationArtifact(callID string, result map[string]any) *assistantartifact.Artifact {
	path := typeutils.StringOfTrimmed(result["path"])
	if !assistantartifact.IsAppPath(path) {
		return nil
	}

	name := typeutils.StringOfTrimmed(result["name"])

	return &assistantartifact.Artifact{
		Kind:   assistantartifact.KindNavigation,
		Status: assistantartifact.StatusReady,
		Title:  assistantartifact.NormalizeTitle(name),
		Payload: map[string]any{
			"path":     path,
			"name":     name,
			"location": typeutils.StringOfTrimmed(result["location"]),
			"page":     typeutils.StringOfTrimmed(result["page"]),
		},
		SourceToolCallID: callID,
	}
}

// NavigationPath is where a navigation artifact goes; "" for any other kind.
func NavigationPath(artifact *assistantartifact.Artifact) string {
	if artifact.Kind != assistantartifact.KindNavigation {
		return ""
	}

	return typeutils.StringOfTrimmed(artifact.Payload["path"])
}

func draftEditArtifact(callID string, result map[string]any) *assistantartifact.Artifact {
	edit, ok := decodeDraftEdit(result["draft"])
	if !ok {
		return nil
	}

	payload, ok := artifactpayload.ToJSONDocument(edit)
	if !ok {
		return nil
	}

	return &assistantartifact.Artifact{
		Kind:             assistantartifact.KindDraftEdit,
		Status:           assistantartifact.StatusReady,
		Title:            assistantartifact.NormalizeTitle(edit.Title()),
		Payload:          payload.Fields,
		SourceToolCallID: callID,
	}
}

func decodeDraftEdit(raw any) (*pagedraft.Edit, bool) {
	if raw == nil {
		return nil, false
	}

	edit := new(pagedraft.Edit)
	if err := jsonutils.Convert(raw, edit); err != nil {
		return nil, false
	}
	if !edit.Action.IsValid() || edit.Surface != edit.Action.Surface() {
		return nil, false
	}

	return edit, true
}

func DraftEditOf(artifact *assistantartifact.Artifact) *pagedraft.Edit {
	if artifact.Kind != assistantartifact.KindDraftEdit {
		return nil
	}

	edit, ok := decodeDraftEdit(artifact.Payload)
	if !ok {
		return nil
	}

	return edit
}

func runDiffTitle(result map[string]any) string {
	side, _ := result["after"].(map[string]any)
	if name := typeutils.StringOfTrimmed(side["reportName"]); name != "" {
		return name + " — what changed"
	}

	return "What changed"
}

func previewArtifact(
	callID string,
	result map[string]any,
	arguments map[string]any,
) *assistantartifact.Artifact {
	rows, _ := result["rows"].([]any)
	payload := map[string]any{
		"name":      typeutils.StringOfTrimmed(result["name"]),
		"dataset":   typeutils.StringOfTrimmed(result["dataset"]),
		"columns":   result["columns"],
		"rowCount":  result["rowCount"],
		"rows":      rows,
		"totals":    result["totals"],
		"truncated": typeutils.BoolOf(result["truncated"]),
		"tool":      toolPreviewReport,
	}
	// A saved or built-in report can be opened whole where reports live; an
	// inline definition has nowhere but its own page.
	if id := typeutils.StringOfTrimmed(arguments["definitionId"]); id != "" {
		payload["definitionId"] = id
	}
	if key := typeutils.StringOfTrimmed(arguments["reportKey"]); key != "" {
		payload["reportKey"] = key
	}
	fitRows(payload, "rows")

	title := typeutils.StringOfTrimmed(result["name"])
	if title == "" {
		title = "Preview of " +
			textcase.HumanizeSnakeCasePhrase(typeutils.StringOfTrimmed(result["dataset"]))
	}

	return &assistantartifact.Artifact{
		Kind:             assistantartifact.KindReportPreview,
		Status:           assistantartifact.StatusReady,
		Title:            assistantartifact.NormalizeTitle(title),
		Payload:          payload,
		SourceToolCallID: callID,
	}
}

func RunArtifact(callID string, result map[string]any) *assistantartifact.Artifact {
	runID := typeutils.StringOfTrimmed(result["runId"])
	if runID == "" {
		return nil
	}

	payload := map[string]any{
		"runId":        runID,
		"reportKey":    typeutils.StringOfTrimmed(result["reportKey"]),
		"definitionId": typeutils.StringOfTrimmed(result["definitionId"]),
		"reportName":   typeutils.StringOfTrimmed(result["reportName"]),
		"status":       typeutils.StringOfTrimmed(result["status"]),
		"finished":     typeutils.BoolOf(result["finished"]),
		"format":       typeutils.StringOfTrimmed(result["format"]),
		"rowCount":     result["rowCount"],
		"truncated":    typeutils.BoolOf(result["truncated"]),
	}

	title := typeutils.StringOfTrimmed(result["reportName"])
	if title == "" {
		title = textcase.HumanizeSnakeCasePhrase(typeutils.StringOfTrimmed(result["reportKey"]))
	}
	if title == "" {
		title = "Report run"
	}

	return &assistantartifact.Artifact{
		Kind: assistantartifact.KindReportRun,
		Status: runStatus(
			typeutils.BoolOf(result["finished"]),
			typeutils.StringOfTrimmed(result["status"]),
		),
		Title:            assistantartifact.NormalizeTitle(title),
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

// EntityCardArtifact shows one record a get tool fetched. A result without
// an id is a view rather than a record (a board, a schedule) and is left to
// the transcript.
//
// The card keeps the record's readable fields, not the record: its id is how
// the Open link is built and is not shown, and its tenancy, version and
// nested structure are the model's business.
func EntityCardArtifact(
	callID, toolName string,
	document artifactpayload.JSONDocument,
) *assistantartifact.Artifact {
	return RecordCardArtifact(&RecordCard{
		CallID:   callID,
		ToolName: toolName,
		Entity:   strings.TrimPrefix(agenttoolports.ReadingOrder(toolName), getToolPrefix),
		Record:   document.Fields,
		Order:    document.KeyOrder(),
	})
}

// RecordCard is one record to draw as a card: the call that read it, the
// kind of record it is, and its fields in the order they were declared.
type RecordCard struct {
	CallID   string
	ToolName string
	Entity   string
	Record   map[string]any
	Order    []string
}

func RecordCardArtifact(card *RecordCard) *assistantartifact.Artifact {
	callID, toolName, entity, result := card.CallID, card.ToolName, card.Entity, card.Record
	if typeutils.StringOfTrimmed(result["id"]) == "" {
		return nil
	}

	payload := map[string]any{
		"display":                     assistantartifact.DisplayVersion,
		artifactpayload.PayloadEntity: entity,
		payloadFields:                 artifactpayload.ProjectRecord(result, card.Order),
		artifactpayload.PayloadTool:   toolName,
		PayloadRecordID:               typeutils.StringOfTrimmed(result["id"]),
	}
	// Where the record opens, from the same registry the app's own links
	// use, so the card leads to the record rather than only describing it.
	if path, ok := productguide.RecordPath(entity, typeutils.StringOfTrimmed(result["id"])); ok {
		payload["path"] = path
	}
	if view := recordviews.RecordView(entity, result); view != nil {
		payload[recordviews.PayloadView] = view
	}
	if artifactpayload.PayloadSize(payload) > assistantartifact.MaxPayloadBytes {
		return nil
	}

	return &assistantartifact.Artifact{
		Kind:             assistantartifact.KindEntityCard,
		Status:           assistantartifact.StatusReady,
		Title:            assistantartifact.NormalizeTitle(entityCardTitle(entity, result)),
		Payload:          payload,
		SourceToolCallID: callID,
	}
}

// entityCardTitle names a record card. A billing queue item is named as a
// biller knows it, by its number and who it bills, as the design titles it;
// any other record by its kind and label.
func entityCardTitle(entity string, result map[string]any) string {
	// A billing queue item and an invoice carry the shipment's pro number
	// too, and a title led by it named the invoice for a load rather than by
	// its own number.
	switch entity {
	case recordviews.ViewBillingQueueItem, recordviews.ViewInvoice:
		number := typeutils.StringOfTrimmed(result["number"])
		billTo := typeutils.StringOfTrimmed(result["billTo"])
		if billTo == "" {
			billTo = typeutils.StringOfTrimmed(result["billToName"])
		}
		if number != "" && billTo != "" {
			return number + " · " + billTo
		}
		if number != "" {
			return number
		}
	}

	return assistantartifact.RecordTitle(entity, assistantartifact.RecordLabel(result))
}

// DraftArtifact views an outbound message proposal as a draft. Only the
// tools in draftSpecs produce one; a dispatch or billing write is decided
// from its card.
func DraftArtifact(proposal *agent.AgentProposal) *assistantartifact.Artifact {
	if proposal == nil {
		return nil
	}
	spec, ok := draftSpecs[proposal.ToolName]
	if !ok {
		return nil
	}

	payload := map[string]any{
		"tool":           proposal.ToolName,
		"arguments":      proposal.ToolParams,
		payloadRationale: proposal.Rationale,
	}
	subject := ""
	if spec.subjectKey != "" {
		subject = typeutils.StringOfTrimmed(proposal.ToolParams[spec.subjectKey])
		payload["subject"] = subject
	}
	if spec.bodyKey != "" {
		payload["body"] = typeutils.StringOfTrimmed(proposal.ToolParams[spec.bodyKey])
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
		Status:     artifactpayload.DraftStatus(proposal.Status),
		Title:      assistantartifact.NormalizeTitle(title),
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

func PlanArtifact(
	plan *agentplan.AgentPlan,
	proposals []*agent.AgentProposal,
) *assistantartifact.Artifact {
	ordered := make([]*agent.AgentProposal, 0, len(proposals))
	for _, proposal := range proposals {
		if proposal != nil && proposal.PlanID != nil && *proposal.PlanID == plan.ID {
			ordered = append(ordered, proposal)
		}
	}
	slices.SortStableFunc(ordered, func(a, b *agent.AgentProposal) int {
		return cmp.Compare(a.PlanStep, b.PlanStep)
	})

	steps := make([]map[string]any, 0, len(ordered))
	for _, proposal := range ordered {
		steps = append(steps, map[string]any{
			"step":            proposal.PlanStep,
			payloadProposalID: proposal.ID.String(),
			payloadToolName:   proposal.ToolName,
			payloadRationale:  proposal.Rationale,
		})
	}

	return &assistantartifact.Artifact{
		Kind:   assistantartifact.KindPlan,
		Status: artifactpayload.PlanStatus(plan.Status),
		Title:  assistantartifact.NormalizeTitle(plan.Title),
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

// fitRows halves the rows under key until the payload is within bound, and
// says so. A preview the tool already capped rarely needs it; a wide one
// with long text cells can.
func fitRows(payload map[string]any, key string) {
	rows, _ := payload[key].([]any)
	for artifactpayload.PayloadSize(payload) > assistantartifact.MaxPayloadBytes && len(rows) > minPreviewRows {
		rows = rows[:len(rows)/2]
		payload[key] = rows
		payload["truncated"] = true
	}
	if len(rows) <= minPreviewRows &&
		artifactpayload.PayloadSize(payload) > assistantartifact.MaxPayloadBytes {
		payload[key] = []any{}
		payload["truncated"] = true
	}
}
