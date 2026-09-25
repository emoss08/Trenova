package aiauditservice

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"strconv"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/aiaudit"
	"github.com/emoss08/trenova/internal/core/domain/audit"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/csvutils"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	// ExportEnvelopeFormat names the JSON export's layout, so a reader can
	// tell this version from any later one.
	ExportEnvelopeFormat = "trenova.ai-audit/v1"
	exportPageSize       = 1000
	exportBufferSize     = 64 * 1024
	hashAlgorithmSigned  = "HMAC-SHA256"
	hashAlgorithmPlain   = "SHA-256"
)

// ErrExportChanged is a trail that lost rows while it was being exported,
// which only the retention sweep can do.
var ErrExportChanged = errors.New("the AI audit trail changed while it was being exported")

// ExportSpec is one export's scope and reader.
type ExportSpec struct {
	Export      *aiaudit.AIAuditExport
	Summary     *repositories.AIAuditEventSummary
	Ceilings    serviceports.FieldCeilings
	RequestedBy string
	// LinkAuditEntries adds the ids of the audit log rows matched to each
	// event by time; only a reader who may read the audit log is given them.
	LinkAuditEntries bool
	GeneratedAt      time.Time
	Heartbeat        Heartbeat
}

// ExportStats describes a written export file.
type ExportStats struct {
	Rows     int64
	Bytes    int64
	SHA256   string
	FirstSeq int64
	LastSeq  int64
	Complete bool
}

// Exporter writes the trail to a CSV or JSON file one page at a time, so an
// export of a million rows never holds more than a page in memory.
type Exporter struct {
	ledger  repositories.AIAuditRepository
	keyring *Keyring
}

func NewExporter(ledger repositories.AIAuditRepository, keyring *Keyring) *Exporter {
	return &Exporter{ledger: ledger, keyring: keyring}
}

// ChainComplete reports whether an export holds an unbroken stretch of the
// chain: nothing filtered out, and every seq between its first and last row.
func ChainComplete(filter *aiaudit.ExportFilter, summary *repositories.AIAuditEventSummary) bool {
	if summary == nil || summary.Count == 0 || !filter.Unfiltered() {
		return false
	}

	return summary.LastSeq-summary.FirstSeq+1 == summary.Count
}

type countingHash struct {
	sink  io.Writer
	sum   hash.Hash
	bytes int64
}

func (c *countingHash) Write(p []byte) (int, error) {
	n, err := c.sink.Write(p)
	if n > 0 {
		_, _ = c.sum.Write(p[:n])
		c.bytes += int64(n)
	}

	return n, err
}

// Write streams the export to w.
func (x *Exporter) Write(ctx context.Context, w io.Writer, spec *ExportSpec) (*ExportStats, error) {
	counter := &countingHash{sink: w, sum: sha256.New()}
	buffered := bufio.NewWriterSize(counter, exportBufferSize)

	stats := &ExportStats{
		FirstSeq: spec.Summary.FirstSeq,
		LastSeq:  spec.Summary.LastSeq,
		Complete: ChainComplete(spec.Export.Filters, spec.Summary),
	}

	var sink rowSink
	switch spec.Export.Format {
	case aiaudit.ExportFormatJSON:
		sink = &jsonSink{w: buffered}
	case aiaudit.ExportFormatCSV:
		sink = &csvSink{w: csv.NewWriter(buffered)}
	default:
		return nil, fmt.Errorf("unknown AI audit export format %q", spec.Export.Format)
	}

	if err := sink.begin(x.header(spec, stats)); err != nil {
		return nil, err
	}

	rows, err := x.stream(ctx, spec, sink)
	if err != nil {
		return nil, err
	}
	if err = sink.end(); err != nil {
		return nil, err
	}
	if err = buffered.Flush(); err != nil {
		return nil, err
	}
	if rows != spec.Summary.Count {
		return nil, fmt.Errorf("%w: expected %d rows, wrote %d", ErrExportChanged,
			spec.Summary.Count, rows)
	}

	stats.Rows = rows
	stats.Bytes = counter.bytes
	stats.SHA256 = hex.EncodeToString(counter.sum.Sum(nil))

	return stats, nil
}

func (x *Exporter) stream(ctx context.Context, spec *ExportSpec, sink rowSink) (int64, error) {
	export := spec.Export
	scope := repositories.AIAuditEventScope{
		TenantInfo: pagination.TenantInfo{
			OrgID: export.OrganizationID,
			BuID:  export.BusinessUnitID,
		},
		From:        export.RangeFrom,
		To:          export.RangeTo,
		SnapshotSeq: export.SnapshotSeq,
		Filter:      export.Filters,
	}

	var written int64
	page := &repositories.ListAIAuditEventPageRequest{Scope: scope, Limit: exportPageSize}
	for {
		if err := ctx.Err(); err != nil {
			return written, err
		}

		events, err := x.ledger.ListPage(ctx, page)
		if err != nil {
			return written, err
		}
		if len(events) == 0 {
			return written, nil
		}

		linked := map[pulid.ID][]*audit.Entry{}
		if spec.LinkAuditEntries {
			if linked, err = Correlate(ctx, x.ledger, scope.TenantInfo, events); err != nil {
				return written, err
			}
		}

		for _, event := range events {
			row := toExportRow(ctx, event, spec.Ceilings, linked[event.ID], spec.LinkAuditEntries)
			if err = sink.row(row); err != nil {
				return written, err
			}
			written++
		}

		last := events[len(events)-1]
		page.AfterOccurred = last.OccurredAt
		page.AfterID = last.ID
		page.HasAfterCursor = true
		beat(spec.Heartbeat, written)

		if len(events) < exportPageSize {
			return written, nil
		}
	}
}

// exportHeader is the JSON export's description of itself.
type exportHeader struct {
	Format      string                `json:"format"`
	ID          string                `json:"id"`
	Tenant      exportTenant          `json:"tenant"`
	GeneratedAt int64                 `json:"generatedAt"`
	GeneratedBy exportRequester       `json:"generatedBy"`
	Range       exportRange           `json:"range"`
	Filters     *aiaudit.ExportFilter `json:"filters"`
	Chain       exportChain           `json:"chain"`
	Columns     []string              `json:"columns"`
}

type exportTenant struct {
	OrganizationID string `json:"organizationId"`
	BusinessUnitID string `json:"businessUnitId"`
}

type exportRequester struct {
	UserID string `json:"userId"`
	Name   string `json:"name,omitempty"`
}

type exportRange struct {
	From int64 `json:"from"`
	To   int64 `json:"to"`
}

type exportChain struct {
	KeyID         string `json:"keyId"`
	Signed        bool   `json:"signed"`
	HashAlgorithm string `json:"hashAlgorithm"`
	FirstSeq      int64  `json:"firstSeq"`
	LastSeq       int64  `json:"lastSeq"`
	SnapshotSeq   int64  `json:"snapshotSeq"`
	Complete      bool   `json:"complete"`
}

func (x *Exporter) header(spec *ExportSpec, stats *ExportStats) *exportHeader {
	export := spec.Export
	algorithm := hashAlgorithmPlain
	if x.keyring.Signed() {
		algorithm = hashAlgorithmSigned
	}

	return &exportHeader{
		Format: ExportEnvelopeFormat,
		ID:     export.ID.String(),
		Tenant: exportTenant{
			OrganizationID: export.OrganizationID.String(),
			BusinessUnitID: export.BusinessUnitID.String(),
		},
		GeneratedAt: spec.GeneratedAt.Unix(),
		GeneratedBy: exportRequester{
			UserID: export.RequestedByUserID.String(),
			Name:   spec.RequestedBy,
		},
		Range:   exportRange{From: export.RangeFrom, To: export.RangeTo},
		Filters: export.Filters,
		Chain: exportChain{
			KeyID:         x.keyring.ActiveKeyID(),
			Signed:        x.keyring.Signed(),
			HashAlgorithm: algorithm,
			FirstSeq:      stats.FirstSeq,
			LastSeq:       stats.LastSeq,
			SnapshotSeq:   export.SnapshotSeq,
			Complete:      stats.Complete,
		},
		Columns: exportColumnNames(),
	}
}

type rowSink interface {
	begin(header *exportHeader) error
	row(row *exportRow) error
	end() error
}

type jsonSink struct {
	w     *bufio.Writer
	count int64
}

func (s *jsonSink) begin(header *exportHeader) error {
	encoded, err := canonicalJSON.Marshal(header)
	if err != nil {
		return err
	}
	if _, err = s.w.WriteString(`{"export":`); err != nil {
		return err
	}
	if _, err = s.w.Write(encoded); err != nil {
		return err
	}
	_, err = s.w.WriteString(`,"events":[`)

	return err
}

func (s *jsonSink) row(row *exportRow) error {
	encoded, err := canonicalJSON.Marshal(row)
	if err != nil {
		return err
	}
	if s.count > 0 {
		if err = s.w.WriteByte(','); err != nil {
			return err
		}
	}
	s.count++
	_, err = s.w.Write(encoded)

	return err
}

func (s *jsonSink) end() error {
	_, err := s.w.WriteString("]}\n")

	return err
}

type csvSink struct {
	w *csv.Writer
}

func (s *csvSink) begin(*exportHeader) error {
	return s.w.Write(exportColumnNames())
}

func (s *csvSink) row(row *exportRow) error {
	record := make([]string, len(exportColumns))
	for i, column := range exportColumns {
		record[i] = csvutils.SafeCell(column.value(row))
	}

	return s.w.Write(record)
}

func (s *csvSink) end() error {
	s.w.Flush()

	return s.w.Error()
}

// exportRow is one event as an export writes it: every column, the
// arguments as this reader may see them, and the chain fields that let a
// holder of the key check the row.
type exportRow struct {
	Seq                    int64          `json:"seq"`
	ID                     string         `json:"id"`
	OccurredAt             int64          `json:"occurredAt"`
	RecordedAt             int64          `json:"recordedAt"`
	Kind                   string         `json:"kind"`
	Outcome                string         `json:"outcome"`
	Purpose                string         `json:"purpose"`
	PrincipalType          string         `json:"principalType"`
	PrincipalID            string         `json:"principalId"`
	OnBehalfOfUserID       string         `json:"onBehalfOfUserId"`
	OnBehalfOfUserName     string         `json:"onBehalfOfUserName"`
	DecidedByUserID        string         `json:"decidedByUserId"`
	DecidedByUserName      string         `json:"decidedByUserName"`
	AgentDefinitionID      string         `json:"agentDefinitionId"`
	AgentDefinitionVersion *int64         `json:"agentDefinitionVersion"`
	AgentName              string         `json:"agentName"`
	OwnerKind              string         `json:"ownerKind"`
	OwnerID                string         `json:"ownerId"`
	RunID                  string         `json:"runId"`
	TurnID                 string         `json:"turnId"`
	ThreadID               string         `json:"threadId"`
	ProposalID             string         `json:"proposalId"`
	PlanID                 string         `json:"planId"`
	DecisionID             string         `json:"decisionId"`
	StepKey                string         `json:"stepKey"`
	CallID                 string         `json:"callId"`
	DelegateCallID         string         `json:"delegateCallId"`
	ParentOwnerID          string         `json:"parentOwnerId"`
	TraceID                string         `json:"traceId"`
	SpanID                 string         `json:"spanId"`
	ProviderID             string         `json:"providerId"`
	ProviderKind           string         `json:"providerKind"`
	Model                  string         `json:"model"`
	Attempt                *int           `json:"attempt"`
	Failover               bool           `json:"failover"`
	InputTokens            int            `json:"inputTokens"`
	OutputTokens           int            `json:"outputTokens"`
	ReasoningTokens        int            `json:"reasoningTokens"`
	CacheReadTokens        int            `json:"cacheReadTokens"`
	CacheWriteTokens       int            `json:"cacheWriteTokens"`
	CostUSD                *string        `json:"costUsd"`
	LatencyMs              *int64         `json:"latencyMs"`
	ToolName               string         `json:"toolName"`
	ToolEffect             string         `json:"toolEffect"`
	EgressClass            string         `json:"egressClass"`
	Tier                   string         `json:"tier"`
	TierSource             string         `json:"tierSource"`
	HeldBy                 []string       `json:"heldBy"`
	Reason                 string         `json:"reason"`
	Arguments              map[string]any `json:"arguments"`
	ArgumentsWithheld      bool           `json:"argumentsWithheld"`
	RedactedPaths          []string       `json:"redactedPaths"`
	ArgumentsTruncated     bool           `json:"argumentsTruncated"`
	ResultSummary          string         `json:"resultSummary"`
	EntityType             string         `json:"entityType"`
	EntityID               string         `json:"entityId"`
	VersionBefore          *int64         `json:"versionBefore"`
	VersionAfter           *int64         `json:"versionAfter"`
	WindowStart            int64          `json:"windowStart"`
	WindowEnd              int64          `json:"windowEnd"`
	Tainted                bool           `json:"tainted"`
	Taint                  any            `json:"taint"`
	ExternalContent        bool           `json:"externalContent"`
	Simulated              bool           `json:"simulated"`
	Reconstructed          bool           `json:"reconstructed"`
	SourceKey              string         `json:"sourceKey"`
	AuditEntryIDs          []string       `json:"auditEntryIds,omitempty"`
	PrevHash               string         `json:"prevHash"`
	Hash                   string         `json:"hash"`
	HashKeyID              string         `json:"hashKeyId"`
	HashVersion            int16          `json:"hashVersion"`
}

func toExportRow(
	ctx context.Context,
	event *aiaudit.AIAuditEvent,
	ceilings serviceports.FieldCeilings,
	linked []*audit.Entry,
	link bool,
) *exportRow {
	arguments := ReaderArguments(ctx, event, ceilings)
	var cost *string
	if event.CostUSD != nil {
		fixed := event.CostUSD.StringFixed(6)
		cost = &fixed
	}

	row := &exportRow{
		Seq:                    event.Seq,
		ID:                     event.ID.String(),
		OccurredAt:             event.OccurredAt,
		RecordedAt:             event.RecordedAt,
		Kind:                   string(event.Kind),
		Outcome:                string(event.Outcome),
		Purpose:                string(event.Purpose),
		PrincipalType:          string(event.PrincipalType),
		PrincipalID:            event.PrincipalID,
		OnBehalfOfUserID:       event.OnBehalfOfUserID.String(),
		OnBehalfOfUserName:     event.OnBehalfOfUserName,
		DecidedByUserID:        event.DecidedByUserID.String(),
		DecidedByUserName:      event.DecidedByUserName,
		AgentDefinitionID:      event.AgentDefinitionID.String(),
		AgentDefinitionVersion: event.AgentDefinitionVersion,
		AgentName:              event.AgentName,
		OwnerKind:              event.OwnerKind,
		OwnerID:                event.OwnerID.String(),
		RunID:                  event.RunID.String(),
		TurnID:                 event.TurnID.String(),
		ThreadID:               event.ThreadID.String(),
		ProposalID:             event.ProposalID.String(),
		PlanID:                 event.PlanID.String(),
		DecisionID:             event.DecisionID.String(),
		StepKey:                event.StepKey,
		CallID:                 event.CallID,
		DelegateCallID:         event.DelegateCallID,
		ParentOwnerID:          event.ParentOwnerID.String(),
		TraceID:                event.TraceID,
		SpanID:                 event.SpanID,
		ProviderID:             event.ProviderID.String(),
		ProviderKind:           event.ProviderKind,
		Model:                  event.Model,
		Attempt:                event.Attempt,
		Failover:               event.Failover,
		InputTokens:            event.InputTokens,
		OutputTokens:           event.OutputTokens,
		ReasoningTokens:        event.ReasoningTokens,
		CacheReadTokens:        event.CacheReadTokens,
		CacheWriteTokens:       event.CacheWriteTokens,
		CostUSD:                cost,
		LatencyMs:              event.LatencyMs,
		ToolName:               event.ToolName,
		ToolEffect:             event.ToolEffect,
		EgressClass:            event.EgressClass,
		Tier:                   event.Tier,
		TierSource:             event.TierSource,
		HeldBy:                 nonNil(event.HeldBy),
		Reason:                 event.Reason,
		Arguments:              arguments,
		ArgumentsWithheld:      withheldAny(event.Arguments, arguments),
		RedactedPaths:          nonNil(event.RedactedPaths),
		ArgumentsTruncated:     event.ArgumentsTruncated,
		ResultSummary:          event.ResultSummary,
		EntityType:             event.EntityType,
		EntityID:               event.EntityID,
		VersionBefore:          event.VersionBefore,
		VersionAfter:           event.VersionAfter,
		WindowStart:            event.WindowStart,
		WindowEnd:              event.WindowEnd,
		Tainted:                event.Tainted,
		ExternalContent:        event.ExternalContent,
		Simulated:              event.Simulated,
		Reconstructed:          event.Reconstructed,
		SourceKey:              event.SourceKey,
		PrevHash:               event.PrevHash,
		Hash:                   event.Hash,
		HashKeyID:              event.HashKeyID,
		HashVersion:            event.HashVersion,
	}
	if event.Taint.Tainted() {
		row.Taint = event.Taint
	}
	if link {
		row.AuditEntryIDs = make([]string, 0, len(linked))
		for _, entry := range linked {
			row.AuditEntryIDs = append(row.AuditEntryIDs, entry.ID.String())
		}
	}

	return row
}

// withheldAny reports whether the reader was shown less than was recorded.
func withheldAny(recorded, shown map[string]any) bool {
	if len(recorded) == 0 {
		return false
	}
	recordedJSON, err := canonicalJSON.Marshal(recorded)
	if err != nil {
		return false
	}
	shownJSON, err := canonicalJSON.Marshal(shown)
	if err != nil {
		return true
	}

	return string(recordedJSON) != string(shownJSON)
}

func nonNil(values []string) []string {
	if values == nil {
		return []string{}
	}

	return values
}

type exportColumn struct {
	name  string
	value func(*exportRow) string
}

func text(value string) string { return value }

func integer(value int64) string { return strconv.FormatInt(value, 10) }

func optionalInt(value *int64) string {
	if value == nil {
		return ""
	}

	return strconv.FormatInt(*value, 10)
}

func boolean(value bool) string { return strconv.FormatBool(value) }

func utc(value int64) string {
	if value <= 0 {
		return ""
	}

	return time.Unix(value, 0).UTC().Format(time.RFC3339)
}

func compact(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case map[string]any:
		if len(typed) == 0 {
			return ""
		}
	case []string:
		if len(typed) == 0 {
			return ""
		}
	}

	encoded, err := canonicalJSON.Marshal(value)
	if err != nil {
		return ""
	}

	return string(encoded)
}

// exportColumns is the CSV column contract. Columns are only ever added at
// the end; a consumer reading by position keeps working.
var exportColumns = []exportColumn{
	{"seq", func(r *exportRow) string { return integer(r.Seq) }},
	{"id", func(r *exportRow) string { return text(r.ID) }},
	{"occurred_at", func(r *exportRow) string { return integer(r.OccurredAt) }},
	{"occurred_at_utc", func(r *exportRow) string { return utc(r.OccurredAt) }},
	{"recorded_at", func(r *exportRow) string { return integer(r.RecordedAt) }},
	{"kind", func(r *exportRow) string { return text(r.Kind) }},
	{"outcome", func(r *exportRow) string { return text(r.Outcome) }},
	{"purpose", func(r *exportRow) string { return text(r.Purpose) }},
	{"principal_type", func(r *exportRow) string { return text(r.PrincipalType) }},
	{"principal_id", func(r *exportRow) string { return text(r.PrincipalID) }},
	{"on_behalf_of_user_id", func(r *exportRow) string { return text(r.OnBehalfOfUserID) }},
	{"on_behalf_of_user_name", func(r *exportRow) string { return text(r.OnBehalfOfUserName) }},
	{"decided_by_user_id", func(r *exportRow) string { return text(r.DecidedByUserID) }},
	{"decided_by_user_name", func(r *exportRow) string { return text(r.DecidedByUserName) }},
	{"agent_definition_id", func(r *exportRow) string { return text(r.AgentDefinitionID) }},
	{"agent_definition_version", func(r *exportRow) string {
		return optionalInt(r.AgentDefinitionVersion)
	}},
	{"agent_name", func(r *exportRow) string { return text(r.AgentName) }},
	{"owner_kind", func(r *exportRow) string { return text(r.OwnerKind) }},
	{"owner_id", func(r *exportRow) string { return text(r.OwnerID) }},
	{"run_id", func(r *exportRow) string { return text(r.RunID) }},
	{"turn_id", func(r *exportRow) string { return text(r.TurnID) }},
	{"thread_id", func(r *exportRow) string { return text(r.ThreadID) }},
	{"proposal_id", func(r *exportRow) string { return text(r.ProposalID) }},
	{"plan_id", func(r *exportRow) string { return text(r.PlanID) }},
	{"decision_id", func(r *exportRow) string { return text(r.DecisionID) }},
	{"step_key", func(r *exportRow) string { return text(r.StepKey) }},
	{"call_id", func(r *exportRow) string { return text(r.CallID) }},
	{"delegate_call_id", func(r *exportRow) string { return text(r.DelegateCallID) }},
	{"parent_owner_id", func(r *exportRow) string { return text(r.ParentOwnerID) }},
	{"trace_id", func(r *exportRow) string { return text(r.TraceID) }},
	{"span_id", func(r *exportRow) string { return text(r.SpanID) }},
	{"provider_id", func(r *exportRow) string { return text(r.ProviderID) }},
	{"provider_kind", func(r *exportRow) string { return text(r.ProviderKind) }},
	{"model", func(r *exportRow) string { return text(r.Model) }},
	{"attempt", func(r *exportRow) string {
		if r.Attempt == nil {
			return ""
		}
		return strconv.Itoa(*r.Attempt)
	}},
	{"failover", func(r *exportRow) string { return boolean(r.Failover) }},
	{"input_tokens", func(r *exportRow) string { return strconv.Itoa(r.InputTokens) }},
	{"output_tokens", func(r *exportRow) string { return strconv.Itoa(r.OutputTokens) }},
	{"reasoning_tokens", func(r *exportRow) string { return strconv.Itoa(r.ReasoningTokens) }},
	{"cache_read_tokens", func(r *exportRow) string { return strconv.Itoa(r.CacheReadTokens) }},
	{"cache_write_tokens", func(r *exportRow) string { return strconv.Itoa(r.CacheWriteTokens) }},
	{"cost_usd", func(r *exportRow) string {
		if r.CostUSD == nil {
			return ""
		}
		return *r.CostUSD
	}},
	{"latency_ms", func(r *exportRow) string { return optionalInt(r.LatencyMs) }},
	{"tool_name", func(r *exportRow) string { return text(r.ToolName) }},
	{"tool_effect", func(r *exportRow) string { return text(r.ToolEffect) }},
	{"egress_class", func(r *exportRow) string { return text(r.EgressClass) }},
	{"tier", func(r *exportRow) string { return text(r.Tier) }},
	{"tier_source", func(r *exportRow) string { return text(r.TierSource) }},
	{"held_by", func(r *exportRow) string { return compact(r.HeldBy) }},
	{"reason", func(r *exportRow) string { return text(r.Reason) }},
	{"arguments", func(r *exportRow) string { return compact(r.Arguments) }},
	{"arguments_withheld", func(r *exportRow) string { return boolean(r.ArgumentsWithheld) }},
	{"redacted_paths", func(r *exportRow) string { return compact(r.RedactedPaths) }},
	{"arguments_truncated", func(r *exportRow) string { return boolean(r.ArgumentsTruncated) }},
	{"result_summary", func(r *exportRow) string { return text(r.ResultSummary) }},
	{"entity_type", func(r *exportRow) string { return text(r.EntityType) }},
	{"entity_id", func(r *exportRow) string { return text(r.EntityID) }},
	{"version_before", func(r *exportRow) string { return optionalInt(r.VersionBefore) }},
	{"version_after", func(r *exportRow) string { return optionalInt(r.VersionAfter) }},
	{"window_start", func(r *exportRow) string { return integer(r.WindowStart) }},
	{"window_end", func(r *exportRow) string { return integer(r.WindowEnd) }},
	{"tainted", func(r *exportRow) string { return boolean(r.Tainted) }},
	{"taint", func(r *exportRow) string { return compact(r.Taint) }},
	{"external_content", func(r *exportRow) string { return boolean(r.ExternalContent) }},
	{"simulated", func(r *exportRow) string { return boolean(r.Simulated) }},
	{"reconstructed", func(r *exportRow) string { return boolean(r.Reconstructed) }},
	{"source_key", func(r *exportRow) string { return text(r.SourceKey) }},
	{"audit_entry_ids", func(r *exportRow) string { return compact(r.AuditEntryIDs) }},
	{"prev_hash", func(r *exportRow) string { return text(r.PrevHash) }},
	{"hash", func(r *exportRow) string { return text(r.Hash) }},
	{"hash_key_id", func(r *exportRow) string { return text(r.HashKeyID) }},
	{"hash_version", func(r *exportRow) string { return strconv.Itoa(int(r.HashVersion)) }},
}

func exportColumnNames() []string {
	names := make([]string, len(exportColumns))
	for i, column := range exportColumns {
		names[i] = column.name
	}

	return names
}
