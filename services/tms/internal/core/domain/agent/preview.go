package agent

import (
	"fmt"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/shopspring/decimal"
)

// PreviewSchemaVersion is the shape of a stored preview. A preview recorded
// under another version is read as it was written and never re-derived.
const PreviewSchemaVersion = 1

const (
	MaxPreviewRecords         = 20
	MaxPreviewFieldsPerRecord = 60
	MaxPreviewValueBytes      = 2 << 10
	MaxPreviewBodyBytes       = 16 << 10
	MaxPreviewBaselineBytes   = 32 << 10
	MaxPreviewSimulationBytes = 8 << 10
	MaxPreviewSummaryRunes    = 500
	MaxPreviewWarnings        = 20
	MaxPreviewReasons         = 20
	MaxPreviewListEntries     = 50
	MaxPreviewLabelRunes      = 200
	MaxPreviewMoneyLines      = 40
	maxSimulationValueRunes   = 120
)

type PreviewCoverage string

const (
	PreviewCoverageFull        = PreviewCoverage("Full")
	PreviewCoveragePartial     = PreviewCoverage("Partial")
	PreviewCoverageUnavailable = PreviewCoverage("Unavailable")
)

func (c PreviewCoverage) IsValid() bool {
	switch c {
	case PreviewCoverageFull, PreviewCoveragePartial, PreviewCoverageUnavailable:
		return true
	default:
		return false
	}
}

type PreviewOperation string

const (
	PreviewOperationCreate  = PreviewOperation("Create")
	PreviewOperationUpdate  = PreviewOperation("Update")
	PreviewOperationDelete  = PreviewOperation("Delete")
	PreviewOperationArchive = PreviewOperation("Archive")
	PreviewOperationSend    = PreviewOperation("Send")
	PreviewOperationRun     = PreviewOperation("Run")
)

func (o PreviewOperation) IsValid() bool {
	switch o {
	case PreviewOperationCreate, PreviewOperationUpdate, PreviewOperationDelete,
		PreviewOperationArchive, PreviewOperationSend, PreviewOperationRun:
		return true
	default:
		return false
	}
}

type MessageChannel string

const (
	MessageChannelEmail   = MessageChannel("Email")
	MessageChannelSMS     = MessageChannel("SMS")
	MessageChannelDash    = MessageChannel("Dash")
	MessageChannelEDI     = MessageChannel("EDI")
	MessageChannelComment = MessageChannel("Comment")
)

func (c MessageChannel) IsValid() bool {
	switch c {
	case MessageChannelEmail, MessageChannelSMS, MessageChannelDash, MessageChannelEDI,
		MessageChannelComment:
		return true
	default:
		return false
	}
}

// PreviewWarningCode is what the client translates; a warning's Message is
// the English fallback.
type PreviewWarningCode string

const (
	PreviewWarningWouldFail           = PreviewWarningCode("would_fail")
	PreviewWarningAlreadyToldCustomer = PreviewWarningCode("already_told_customer")
	PreviewWarningDriverUnreachable   = PreviewWarningCode("driver_unreachable")
	PreviewWarningDependsOnStep       = PreviewWarningCode("depends_on_step")
	PreviewWarningTargetChanged       = PreviewWarningCode("target_changed")
	PreviewWarningRecordMissing       = PreviewWarningCode("record_missing")
	PreviewWarningToolRemoved         = PreviewWarningCode("tool_removed")
	PreviewWarningPreviewFailed       = PreviewWarningCode("preview_failed")
	PreviewWarningWithheld            = PreviewWarningCode("withheld")
	PreviewWarningSensitiveContent    = PreviewWarningCode("sensitive_content")
	PreviewWarningRetargetRefused     = PreviewWarningCode("retarget_refused")
	PreviewWarningUnpinned            = PreviewWarningCode("unpinned")
)

func AllPreviewWarningCodes() []PreviewWarningCode {
	return []PreviewWarningCode{
		PreviewWarningWouldFail,
		PreviewWarningAlreadyToldCustomer,
		PreviewWarningDriverUnreachable,
		PreviewWarningDependsOnStep,
		PreviewWarningTargetChanged,
		PreviewWarningRecordMissing,
		PreviewWarningToolRemoved,
		PreviewWarningPreviewFailed,
		PreviewWarningWithheld,
		PreviewWarningSensitiveContent,
		PreviewWarningRetargetRefused,
		PreviewWarningUnpinned,
	}
}

type PreviewWarning struct {
	Code    PreviewWarningCode `json:"code"`
	Args    []string           `json:"args,omitempty"`
	Message string             `json:"message"`
	// Reasons are the problems a would_fail warning is made of, one per
	// rule the write breaks, so a person can be told each and change the
	// value it names. Message keeps the whole refusal as it was worded.
	Reasons []PreviewReason `json:"reasons,omitempty"`
}

// PreviewReason is one rule a write would break. Field is the path the rule
// names in the record's own terms (bol, moves[0].stops[1].locationId) and
// Label that field in words; both are empty for a refusal of the whole
// write. Param is the path of the call's parameter that carries the field
// (shipment.bol), empty when the call does not carry it, so a person can be
// offered that value to change.
type PreviewReason struct {
	Field   string `json:"field,omitempty"`
	Label   string `json:"label,omitempty"`
	Message string `json:"message"`
	Param   string `json:"param,omitempty"`
}

// ReasonLines are the warning's reasons in plain words, one a line: the
// field's label and the rule's message, and the parameter the field rides
// in where there is one. A warning without reasons is its message alone.
func (w *PreviewWarning) ReasonLines() []string {
	if w == nil {
		return nil
	}
	if len(w.Reasons) == 0 {
		return []string{strings.TrimSpace(w.Message)}
	}

	lines := make([]string, 0, len(w.Reasons))
	for i := range w.Reasons {
		reason := &w.Reasons[i]
		var b strings.Builder
		if reason.Label != "" {
			b.WriteString(reason.Label)
			b.WriteString(": ")
		}
		b.WriteString(reason.Message)
		if reason.Param != "" {
			b.WriteString(" (parameter ")
			b.WriteString(reason.Param)
			b.WriteByte(')')
		}
		lines = append(lines, b.String())
	}

	return lines
}

// PreviewRef is a record a value points at: the id a tool holds, resolved to
// the words a person knows it by once the preview is built.
type PreviewRef struct {
	Resource permission.Resource `json:"resource"`
	ID       pulid.ID            `json:"id"`
	Label    string              `json:"label,omitempty"`
	Record   *RecordRef          `json:"record,omitempty"`
	Withheld bool                `json:"withheld,omitempty"`
}

// PreviewFieldChange is one value a write would set. Before is absent on a
// create and After on a delete. ProposedBefore is what Before was when the
// change was proposed, kept only when it has moved since.
type PreviewFieldChange struct {
	Path                 string                        `json:"path"`
	Label                string                        `json:"label"`
	Type                 assistantartifact.DisplayType `json:"type"`
	Before               any                           `json:"before"`
	After                any                           `json:"after"`
	BeforeRef            *PreviewRef                   `json:"beforeRef,omitempty"`
	AfterRef             *PreviewRef                   `json:"afterRef,omitempty"`
	Sensitivity          permission.FieldSensitivity   `json:"sensitivity,omitempty"`
	Withheld             bool                          `json:"withheld,omitempty"`
	Volatile             bool                          `json:"volatile,omitempty"`
	Truncated            bool                          `json:"truncated,omitempty"`
	ChangedSinceProposed bool                          `json:"changedSinceProposed,omitempty"`
	ProposedBefore       any                           `json:"proposedBefore,omitempty"`
	ProjectedFromStep    int                           `json:"projectedFromStep,omitempty"`
}

// MessagePreview is what a write would send, rendered as it would go out.
type MessagePreview struct {
	Channel           MessageChannel `json:"channel"`
	From              string         `json:"from,omitempty"`
	To                []string       `json:"to,omitempty"`
	Cc                []string       `json:"cc,omitempty"`
	Bcc               []string       `json:"bcc,omitempty"`
	Attachments       []string       `json:"attachments,omitempty"`
	Subject           string         `json:"subject,omitempty"`
	Body              string         `json:"body,omitempty"`
	BodyTruncated     bool           `json:"bodyTruncated,omitempty"`
	Visibility        string         `json:"visibility,omitempty"`
	Cadence           string         `json:"cadence,omitempty"`
	TemplateVersionID pulid.ID       `json:"templateVersionId,omitempty"`
}

type MoneyLine struct {
	Label  string              `json:"label"`
	Before decimal.NullDecimal `json:"before"`
	After  decimal.NullDecimal `json:"after"`
}

// MoneyPreview is an amount a write would move, line by line with its total
// before and after. Sensitivity is that of the least visible figure in it.
type MoneyPreview struct {
	Currency    string                      `json:"currency"`
	Lines       []MoneyLine                 `json:"lines,omitempty"`
	TotalBefore decimal.NullDecimal         `json:"totalBefore"`
	TotalAfter  decimal.NullDecimal         `json:"totalAfter"`
	Delta       decimal.NullDecimal         `json:"delta"`
	Sensitivity permission.FieldSensitivity `json:"sensitivity,omitempty"`
	Withheld    bool                        `json:"withheld,omitempty"`
}

// RecordChange is what a write would do to one record.
type RecordChange struct {
	Resource      permission.Resource  `json:"resource"`
	Record        *RecordRef           `json:"record,omitempty"`
	EntityID      pulid.ID             `json:"entityId,omitempty"`
	Label         string               `json:"label,omitempty"`
	Operation     PreviewOperation     `json:"operation"`
	Version       *int64               `json:"version,omitempty"`
	Withheld      bool                 `json:"withheld,omitempty"`
	DependsOnStep int                  `json:"dependsOnStep,omitempty"`
	Fields        []PreviewFieldChange `json:"fields,omitempty"`
	OmittedFields int                  `json:"omittedFields,omitempty"`
	Message       *MessagePreview      `json:"message,omitempty"`
	Money         *MoneyPreview        `json:"money,omitempty"`
}

// ToolPreview is what a tool says its write would do: no viewer, no digest
// and no staleness, which the preview service adds.
type ToolPreview struct {
	Summary        string           `json:"summary"`
	Changes        []RecordChange   `json:"changes,omitempty"`
	Warnings       []PreviewWarning `json:"warnings,omitempty"`
	Partial        bool             `json:"partial,omitempty"`
	OmittedRecords int              `json:"omittedRecords,omitempty"`
}

// PreviewStaleness compares the record a proposal pinned with the record as
// it is now.
type PreviewStaleness struct {
	Pinned          bool  `json:"pinned"`
	ProposedVersion int64 `json:"proposedVersion"`
	CurrentVersion  int64 `json:"currentVersion"`
	Missing         bool  `json:"missing,omitempty"`
}

func (s *PreviewStaleness) Stale() bool {
	return s != nil && s.Pinned && (s.Missing || s.CurrentVersion != s.ProposedVersion)
}

// ProposalPreview is what the preview service serves and records: a tool's
// preview for one reader, with its coverage, its staleness and the digest of
// what that reader was shown.
type ProposalPreview struct {
	Schema         int               `json:"schema"`
	ProposalID     pulid.ID          `json:"proposalId"`
	Tool           string            `json:"tool"`
	Summary        string            `json:"summary"`
	Coverage       PreviewCoverage   `json:"coverage"`
	Changes        []RecordChange    `json:"changes,omitempty"`
	Warnings       []PreviewWarning  `json:"warnings,omitempty"`
	Staleness      *PreviewStaleness `json:"staleness,omitempty"`
	TargetVersion  *int64            `json:"targetVersion,omitempty"`
	WithheldCount  int               `json:"withheldCount"`
	OmittedRecords int               `json:"omittedRecords,omitempty"`
	Recorded       bool              `json:"recorded,omitempty"`
	ComputedAt     int64             `json:"computedAt"`
	Digest         string            `json:"digest,omitempty"`
}

func (p *ProposalPreview) IsStale() bool {
	return p != nil && p.Staleness.Stale()
}

func (p *ProposalPreview) HasWarning(code PreviewWarningCode) bool {
	if p == nil {
		return false
	}

	return findWarning(p.Warnings, code) != nil
}

// Refusal is the would_fail warning of a tool's preview: what the write
// would be refused over, as it stands. Nil when it would go through.
func (t *ToolPreview) Refusal() *PreviewWarning {
	if t == nil {
		return nil
	}

	return findWarning(t.Warnings, PreviewWarningWouldFail)
}

func findWarning(warnings []PreviewWarning, code PreviewWarningCode) *PreviewWarning {
	for i := range warnings {
		if warnings[i].Code == code {
			return &warnings[i]
		}
	}

	return nil
}

// AddWarning appends a warning once per code and arguments, within bounds.
func (p *ProposalPreview) AddWarning(warning PreviewWarning) {
	p.Warnings = appendWarning(p.Warnings, warning)
}

func (t *ToolPreview) AddWarning(warning PreviewWarning) {
	t.Warnings = appendWarning(t.Warnings, warning)
}

func appendWarning(warnings []PreviewWarning, warning PreviewWarning) []PreviewWarning {
	for i := range warnings {
		if warnings[i].Code == warning.Code &&
			strings.Join(warnings[i].Args, "\x00") == strings.Join(warning.Args, "\x00") {
			return warnings
		}
	}
	if len(warnings) >= MaxPreviewWarnings {
		return warnings
	}

	return append(warnings, warning)
}

// previewDigestDocument is what a digest covers: the proposal, the
// parameters it would run with and what the reader was shown. The instant it
// was computed, the digest itself and every volatile value are left out, so
// the same world read twice digests alike.
type previewDigestDocument struct {
	Schema        int             `json:"schema"`
	ProposalID    pulid.ID        `json:"proposalId"`
	Tool          string          `json:"tool"`
	Params        map[string]any  `json:"params"`
	Coverage      PreviewCoverage `json:"coverage"`
	WithheldCount int             `json:"withheldCount"`
	TargetVersion *int64          `json:"targetVersion"`
	Changes       []RecordChange  `json:"changes"`
}

// ComputeDigest is the SHA-256 of the preview's canonical form together with
// the parameters it was computed for.
func (p *ProposalPreview) ComputeDigest(params map[string]any) (string, error) {
	changes := make([]RecordChange, len(p.Changes))
	for i := range p.Changes {
		changes[i] = p.Changes[i]
		changes[i].Fields = stableFields(p.Changes[i].Fields)
	}

	digest, err := jsonutils.CanonicalDigest(&previewDigestDocument{
		Schema:        p.Schema,
		ProposalID:    p.ProposalID,
		Tool:          p.Tool,
		Params:        params,
		Coverage:      p.Coverage,
		WithheldCount: p.WithheldCount,
		TargetVersion: p.TargetVersion,
		Changes:       changes,
	})
	if err != nil {
		return "", fmt.Errorf("digest proposal preview: %w", err)
	}

	return digest, nil
}

func stableFields(fields []PreviewFieldChange) []PreviewFieldChange {
	stable := make([]PreviewFieldChange, 0, len(fields))
	for i := range fields {
		if !fields[i].Volatile {
			stable = append(stable, fields[i])
		}
	}

	return stable
}

// PlanPreviewDigest covers every step's digest in plan order.
func PlanPreviewDigest(planID pulid.ID, stepDigests []string) (string, error) {
	digest, err := jsonutils.CanonicalDigest(map[string]any{
		"planId": planID.String(),
		"steps":  stepDigests,
	})
	if err != nil {
		return "", fmt.Errorf("digest plan preview: %w", err)
	}

	return digest, nil
}

// Bounded is the preview cut to what is kept and served: at most
// MaxPreviewRecords records of MaxPreviewFieldsPerRecord fields, each value
// at most MaxPreviewValueBytes, a message body at most MaxPreviewBodyBytes.
// A value that was cut says so. Nil stays nil.
func (t *ToolPreview) Bounded() *ToolPreview {
	if t == nil {
		return nil
	}

	bounded := &ToolPreview{
		Summary: stringutils.TruncateRunes(
			strings.TrimSpace(t.Summary),
			MaxPreviewSummaryRunes,
		),
		Partial:        t.Partial,
		OmittedRecords: t.OmittedRecords,
	}

	records := t.Changes
	if len(records) > MaxPreviewRecords {
		bounded.OmittedRecords += len(records) - MaxPreviewRecords
		bounded.Partial = true
		records = records[:MaxPreviewRecords]
	}
	if len(records) > 0 {
		bounded.Changes = make([]RecordChange, 0, len(records))
		for i := range records {
			bounded.Changes = append(bounded.Changes, records[i].bounded())
		}
	}

	for i := range t.Warnings {
		bounded.Warnings = appendWarning(bounded.Warnings, boundedWarning(t.Warnings[i]))
	}

	return bounded
}

func (c *RecordChange) bounded() RecordChange {
	out := *c
	out.Label = stringutils.TruncateRunes(strings.TrimSpace(c.Label), MaxPreviewLabelRunes)

	fields := c.Fields
	if len(fields) > MaxPreviewFieldsPerRecord {
		out.OmittedFields += len(fields) - MaxPreviewFieldsPerRecord
		fields = fields[:MaxPreviewFieldsPerRecord]
	}
	out.Fields = nil
	if len(fields) > 0 {
		out.Fields = make([]PreviewFieldChange, 0, len(fields))
		for i := range fields {
			out.Fields = append(out.Fields, fields[i].bounded())
		}
	}

	if c.Message != nil {
		message := c.Message.bounded()
		out.Message = &message
	}
	if c.Money != nil {
		money := *c.Money
		if len(money.Lines) > MaxPreviewMoneyLines {
			money.Lines = money.Lines[:MaxPreviewMoneyLines]
		}
		money.Lines = append([]MoneyLine(nil), money.Lines...)
		out.Money = &money
	}

	return out
}

func (f *PreviewFieldChange) bounded() PreviewFieldChange {
	out := *f
	out.Label = stringutils.TruncateRunes(strings.TrimSpace(f.Label), MaxPreviewLabelRunes)

	var cut bool
	out.Before, cut = BoundPreviewValue(f.Before)
	out.Truncated = out.Truncated || cut
	out.After, cut = BoundPreviewValue(f.After)
	out.Truncated = out.Truncated || cut
	out.ProposedBefore, cut = BoundPreviewValue(f.ProposedBefore)
	out.Truncated = out.Truncated || cut

	return out
}

func (m *MessagePreview) bounded() MessagePreview {
	out := *m
	out.To = boundedList(m.To)
	out.Cc = boundedList(m.Cc)
	out.Bcc = boundedList(m.Bcc)
	out.Attachments = boundedList(m.Attachments)
	out.Subject = stringutils.TruncateRunes(m.Subject, MaxPreviewLabelRunes)
	if len(m.Body) > MaxPreviewBodyBytes {
		out.Body = stringutils.TruncateBytes(m.Body, MaxPreviewBodyBytes)
		out.BodyTruncated = true
	}

	return out
}

func boundedList(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	if len(values) > MaxPreviewListEntries {
		values = values[:MaxPreviewListEntries]
	}

	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, stringutils.TruncateRunes(value, MaxPreviewLabelRunes))
	}

	return out
}

func boundedWarning(warning PreviewWarning) PreviewWarning {
	out := PreviewWarning{
		Code:    warning.Code,
		Message: stringutils.TruncateRunes(warning.Message, MaxPreviewSummaryRunes),
	}
	if len(warning.Args) > 0 {
		out.Args = boundedList(warning.Args)
	}
	if len(warning.Reasons) > 0 {
		reasons := warning.Reasons
		if len(reasons) > MaxPreviewReasons {
			reasons = reasons[:MaxPreviewReasons]
		}
		out.Reasons = make([]PreviewReason, 0, len(reasons))
		for _, reason := range reasons {
			out.Reasons = append(out.Reasons, PreviewReason{
				Field:   stringutils.TruncateRunes(reason.Field, MaxPreviewLabelRunes),
				Label:   stringutils.TruncateRunes(reason.Label, MaxPreviewLabelRunes),
				Message: stringutils.TruncateRunes(reason.Message, MaxPreviewSummaryRunes),
				Param:   stringutils.TruncateRunes(reason.Param, MaxPreviewLabelRunes),
			})
		}
	}

	return out
}

// BoundPreviewValue keeps a value within MaxPreviewValueBytes: a long string
// is ellipsized, and a list or object whose encoding is too long becomes its
// ellipsized encoding. The second result says it was cut.
func BoundPreviewValue(value any) (any, bool) {
	switch typed := value.(type) {
	case nil, bool, float64, float32, int, int32, int64:
		return typed, false
	case string:
		if len(typed) <= MaxPreviewValueBytes {
			return typed, false
		}

		return ellipsizeBytes(typed, MaxPreviewValueBytes), true
	default:
		encoded, err := sonic.MarshalString(typed)
		if err != nil {
			return fmt.Sprint(typed), false
		}
		if len(encoded) <= MaxPreviewValueBytes {
			return typed, false
		}

		return ellipsizeBytes(encoded, MaxPreviewValueBytes), true
	}
}

func ellipsizeBytes(value string, limit int) string {
	const ellipsis = "…"

	return stringutils.TruncateBytes(value, limit-len(ellipsis)) + ellipsis
}

// Simulation is the preview as the runtime's simulation reads it: the
// summary, then one line per value, message and amount, cut to
// MaxPreviewSimulationBytes.
func (t *ToolPreview) Simulation() *ToolSimulation {
	if t == nil {
		return nil
	}

	simulation := &ToolSimulation{Summary: t.Summary, Previewed: true}
	size := len(simulation.Summary)
	multiple := len(t.Changes) > 1
	for i := range t.Changes {
		for _, change := range t.Changes[i].simulationChanges(multiple) {
			size += len(change.Field) + len(change.From) + len(change.To) + 32
			if size > MaxPreviewSimulationBytes {
				return simulation
			}
			simulation.Changes = append(simulation.Changes, change)
		}
	}

	return simulation
}

func (c *RecordChange) simulationChanges(prefixed bool) []FieldChange {
	prefix := ""
	if prefixed && c.Label != "" {
		prefix = c.Label + ": "
	}

	changes := make([]FieldChange, 0, len(c.Fields)+4)
	if c.Operation != PreviewOperationUpdate && c.Operation != PreviewOperationRun &&
		c.Operation != "" {
		changes = append(changes, FieldChange{
			Field: prefix + "operation",
			To:    strings.ToLower(string(c.Operation)),
		})
	}
	for i := range c.Fields {
		field := &c.Fields[i]
		if field.Withheld || field.Sensitivity == permission.SensitivityConfidential {
			continue
		}
		label := field.Label
		if label == "" {
			label = field.Path
		}
		changes = append(changes, FieldChange{
			Field: prefix + label,
			From:  refText(field.BeforeRef, field.Before),
			To:    stringutils.WithDefault(refText(field.AfterRef, field.After), "nothing"),
		})
	}
	if message := c.Message; message != nil {
		if len(message.To) > 0 {
			changes = append(changes, FieldChange{
				Field: prefix + "to",
				To:    strings.Join(message.To, ", "),
			})
		}
		if message.Subject != "" {
			changes = append(changes, FieldChange{Field: prefix + "subject", To: message.Subject})
		}
		if message.Body != "" {
			changes = append(changes, FieldChange{
				Field: prefix + "body",
				To:    stringutils.Ellipsize(message.Body, maxSimulationValueRunes),
			})
		}
	}
	if money := c.Money; money != nil && !money.Withheld {
		for _, line := range money.Lines {
			changes = append(changes, FieldChange{
				Field: prefix + line.Label,
				From:  nullDecimalText(line.Before),
				To:    nullDecimalText(line.After),
			})
		}
		if money.TotalAfter.Valid {
			changes = append(changes, FieldChange{
				Field: prefix + "total",
				From:  nullDecimalText(money.TotalBefore),
				To:    nullDecimalText(money.TotalAfter),
			})
		}
	}

	return changes
}

func refText(ref *PreviewRef, value any) string {
	if ref != nil && ref.Label != "" && !ref.Withheld {
		return ref.Label
	}

	return PreviewValueText(value)
}

func nullDecimalText(value decimal.NullDecimal) string {
	if !value.Valid {
		return ""
	}

	return value.Decimal.String()
}

// PreviewValueText writes a value in words for a line of text: nothing as
// "nothing", a list or object as its encoding, all of it short.
func PreviewValueText(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		if strings.TrimSpace(typed) == "" {
			return "nothing"
		}

		return stringutils.Ellipsize(typed, maxSimulationValueRunes)
	case bool, int, int32, int64, float32, float64:
		return fmt.Sprint(typed)
	default:
		encoded, err := sonic.MarshalString(typed)
		if err != nil {
			return fmt.Sprint(typed)
		}

		return stringutils.Ellipsize(encoded, maxSimulationValueRunes)
	}
}

// PlanStepPreview is one pending step of a plan and what it would do.
type PlanStepPreview struct {
	ProposalID pulid.ID         `json:"proposalId"`
	Step       int              `json:"step"`
	Preview    *ProposalPreview `json:"preview"`
}

// PlanPreview is every pending step of a plan, projected in order: a step
// that follows another on the same record starts from what that step leaves.
// Digest covers every step's digest in order, and the plan is stale when any
// step is.
type PlanPreview struct {
	PlanID        pulid.ID          `json:"planId"`
	Steps         []PlanStepPreview `json:"steps"`
	Digest        string            `json:"digest"`
	Stale         bool              `json:"stale"`
	WithheldCount int               `json:"withheldCount"`
	ComputedAt    int64             `json:"computedAt"`
}

// StepDigests are the steps' digests in plan order.
func (p *PlanPreview) StepDigests() []string {
	if p == nil {
		return nil
	}

	digests := make([]string, 0, len(p.Steps))
	for i := range p.Steps {
		digest := ""
		if p.Steps[i].Preview != nil {
			digest = p.Steps[i].Preview.Digest
		}
		digests = append(digests, digest)
	}

	return digests
}

// StepFor is the preview of the step that is the given proposal.
func (p *PlanPreview) StepFor(proposalID pulid.ID) *ProposalPreview {
	if p == nil {
		return nil
	}
	for i := range p.Steps {
		if p.Steps[i].ProposalID == proposalID {
			return p.Steps[i].Preview
		}
	}

	return nil
}

// EncodedPreviewSize is the size of a value's encoding, for the byte bounds
// a stored preview keeps to.
func EncodedPreviewSize(value any) int {
	encoded, err := sonic.Marshal(value)
	if err != nil {
		return 0
	}

	return len(encoded)
}
