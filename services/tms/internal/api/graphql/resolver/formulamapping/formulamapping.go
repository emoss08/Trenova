package formulamapping

import (
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentstarter"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/rating/app/tariffverdict"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
)

func Wire(value any) (rules.RawJSON, error) {
	encoded, err := sonic.Marshal(value)
	if err != nil {
		return nil, err
	}
	return rules.RawJSON(encoded), nil
}

func WireList[T any](items []T) (rules.RawJSON, error) {
	if items == nil {
		items = []T{}
	}
	return Wire(items)
}

func InputValues(values map[string]any) (tarifftypes.Inputs, error) {
	if len(values) == 0 {
		return nil, nil
	}
	inputs := make(tarifftypes.Inputs, len(values))
	for name, value := range values {
		encoded, err := sonic.Marshal(value)
		if err != nil {
			return nil, errortypes.NewValidationError(
				"inputs",
				errortypes.ErrInvalid,
				"Input values must be JSON values",
			)
		}
		inputs[name] = rules.RawJSON(encoded)
	}
	return inputs, nil
}

func ParseOptionalID(raw *string, field, message string) (pulid.ID, error) {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return pulid.Nil, nil
	}
	id, err := pulid.Parse(strings.TrimSpace(*raw))
	if err != nil {
		return pulid.Nil, errortypes.NewValidationError(field, errortypes.ErrInvalid, message)
	}
	return id, nil
}

func Changes(changes map[string]jsonutils.FieldChange) map[string]any {
	out := make(map[string]any, len(changes))
	for path, change := range changes {
		out[path] = change
	}
	return out
}

func VersionDiff(
	fromVersion, toVersion int64,
	changes map[string]jsonutils.FieldChange,
	changeCount int,
) *gqlmodel.FormulaVersionDiff {
	return &gqlmodel.FormulaVersionDiff{
		FromVersion: int(fromVersion),
		ToVersion:   int(toVersion),
		Changes:     Changes(changes),
		ChangeCount: changeCount,
	}
}

func Imports(imports []tarifftypes.Import) []*tarifftypes.Import {
	out := make([]*tarifftypes.Import, 0, len(imports))
	for i := range imports {
		out = append(out, &imports[i])
	}
	return out
}

type ReviewDiffParams struct {
	HasApprovedBase  bool
	BaseVersion      int64
	CurrentVersion   int64
	BaseSource       string
	CurrentSource    string
	BaseImports      []tarifftypes.Import
	CurrentImports   []tarifftypes.Import
	Changes          map[string]jsonutils.FieldChange
	ChangeCount      int
	PendingSchedules []*gqlmodel.FormulaScheduledWindow
}

func ReviewDiff(p *ReviewDiffParams) *gqlmodel.FormulaReviewDiff {
	pending := p.PendingSchedules
	if pending == nil {
		pending = []*gqlmodel.FormulaScheduledWindow{}
	}
	return &gqlmodel.FormulaReviewDiff{
		HasApprovedBase:  p.HasApprovedBase,
		BaseVersion:      int(p.BaseVersion),
		CurrentVersion:   int(p.CurrentVersion),
		BaseSource:       p.BaseSource,
		CurrentSource:    p.CurrentSource,
		BaseImports:      Imports(p.BaseImports),
		CurrentImports:   Imports(p.CurrentImports),
		Changes:          Changes(p.Changes),
		ChangeCount:      p.ChangeCount,
		PendingSchedules: pending,
	}
}

func ScheduledWindow(versionNumber, effectiveFrom int64) *gqlmodel.FormulaScheduledWindow {
	return &gqlmodel.FormulaScheduledWindow{
		VersionNumber: int(versionNumber),
		EffectiveFrom: int(effectiveFrom),
	}
}

func ReadinessCheck(key, label, status, detail string) *gqlmodel.FormulaReadinessCheck {
	return &gqlmodel.FormulaReadinessCheck{
		Key:    key,
		Label:  label,
		Status: gqlmodel.FormulaReadinessStatus(status),
		Detail: detail,
	}
}

type ReadinessParams struct {
	CanSubmit        bool
	CanApprove       bool
	Checks           []*gqlmodel.FormulaReadinessCheck
	TestTotal        int
	TestPassed       int
	TestFailing      []string
	Coverage         *tariffverdict.CoverageSummary
	PendingSchedules []*gqlmodel.FormulaScheduledWindow
}

func Readiness(p *ReadinessParams) *gqlmodel.FormulaReadiness {
	failing := p.TestFailing
	if failing == nil {
		failing = []string{}
	}
	pending := p.PendingSchedules
	if pending == nil {
		pending = []*gqlmodel.FormulaScheduledWindow{}
	}
	return &gqlmodel.FormulaReadiness{
		CanSubmit:        p.CanSubmit,
		CanApprove:       p.CanApprove,
		Checks:           p.Checks,
		TestTotal:        p.TestTotal,
		TestPassed:       p.TestPassed,
		TestFailing:      failing,
		Coverage:         Coverage(p.Coverage),
		PendingSchedules: pending,
	}
}

func Coverage(summary *tariffverdict.CoverageSummary) *gqlmodel.FormulaCoverage {
	if summary == nil {
		return nil
	}
	unreached := make([]*gqlmodel.FormulaUnreachedRegion, 0, len(summary.Unreached))
	for i := range summary.Unreached {
		region := &summary.Unreached[i]
		unreached = append(unreached, &gqlmodel.FormulaUnreachedRegion{
			Kind:  string(region.Kind),
			Range: Range(&region.Range),
			At: &gqlmodel.FormulaSourcePosition{
				Line:   region.At.Line,
				Column: region.At.Column,
			},
			Excerpt: region.Excerpt,
		})
	}
	return &gqlmodel.FormulaCoverage{
		Reached:   summary.Reached,
		Total:     summary.Total,
		Unreached: unreached,
		More:      summary.More,
	}
}

func Range(r *rules.TariffRange) *gqlmodel.FormulaSourceRange {
	if r == nil {
		return nil
	}
	return &gqlmodel.FormulaSourceRange{From: r.From, To: r.To}
}

type ReviewerParams struct {
	ID            pulid.ID
	Name          string
	Username      string
	ProfilePicURL string
}

func Reviewer(p ReviewerParams) *gqlmodel.FormulaReviewer {
	return &gqlmodel.FormulaReviewer{
		ID:            p.ID.String(),
		Name:          p.Name,
		Username:      p.Username,
		ProfilePicURL: gqlscalar.NonEmptyPtr(p.ProfilePicURL),
	}
}

func PageThread(opened *assistantports.PageThread) *gqlmodel.PageAssistantThread {
	if opened == nil {
		return nil
	}
	out := &gqlmodel.PageAssistantThread{
		Thread: &gqlmodel.PageAssistantConversation{},
		Agent:  pageAgent(&opened.Agent),
	}
	thread := opened.Thread
	if thread == nil {
		return out
	}
	out.Thread = &gqlmodel.PageAssistantConversation{
		ID:                   thread.ID.String(),
		BusinessUnitID:       thread.BusinessUnitID.String(),
		OrganizationID:       thread.OrganizationID.String(),
		UserID:               thread.UserID.String(),
		AgentDefinitionID:    thread.AgentDefinitionID.String(),
		Title:                thread.Title,
		Status:               string(thread.Status),
		LastMessageAt:        int(thread.LastMessageAt),
		LastReadAt:           int(thread.LastReadAt),
		PreferredProviderID:  gqlid.IDPtr(thread.PreferredProviderID),
		Origin:               string(thread.Origin),
		Pinned:               thread.Pinned,
		SubjectType:          string(thread.SubjectType),
		SubjectID:            gqlid.IDPtr(thread.SubjectID),
		CanContinue:          thread.CanContinue,
		CannotContinueReason: gqlscalar.NonEmptyPtr(string(thread.CannotContinueReason)),
		TaintedAt:            gqlscalar.IntPtr(thread.TaintedAt),
		Version:              int(thread.Version),
		CreatedAt:            int(thread.CreatedAt),
		UpdatedAt:            int(thread.UpdatedAt),
	}
	return out
}

func pageAgent(agent *assistantports.PageAgent) *gqlmodel.PageAssistantAgent {
	out := &gqlmodel.PageAssistantAgent{
		ID:          agent.ID.String(),
		Name:        agent.Name,
		Description: agent.Description,
		Icon:        agent.Icon,
		Accent:      agent.Accent,
		SystemKey:   agent.SystemKey,
		ToolNames:   agent.ToolNames,
	}
	if out.ToolNames == nil {
		out.ToolNames = []string{}
	}
	if agent.Template != "" {
		template := agent.Template
		out.Template = &template
	}
	for i := range agent.Starters {
		out.Starters = append(out.Starters, &agent.Starters[i])
	}
	if out.Starters == nil {
		out.Starters = []*agentstarter.Starter{}
	}
	return out
}
