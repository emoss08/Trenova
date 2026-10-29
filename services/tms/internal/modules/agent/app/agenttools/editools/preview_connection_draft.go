package editools

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolguard"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediconnectiondraftports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/lading"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	fieldName   = "name"
	fieldStatus = "status"
)

var (
	draftPartnerFields = []string{
		"code", fieldName, "kind", fieldStatus, "customerId", "contactName", "contactEmail",
		"contactPhone", "timezone", "country", "enabledForInbound", "enabledForOutbound",
	}
	draftTransportFields = []string{fieldName, "method", fieldStatus, "config"}
	draftProjectFields   = []string{"files", "samples"}
	draftMappingFields   = []string{
		"entityType",
		"sourceId",
		"sourceLabel",
		"targetId",
		"targetLabel",
	}
	draftControlNumberFields = []string{"kind", "nextValue"}
)

func (t *ApplyConnectionDraftTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agentpreview.ToolPreview, error) {
	if err := toolguard.GuardPreview(t, &params); err != nil {
		return nil, err
	}

	summary := "Would set up an EDI connection from the partner's guide."
	input, plan, err := t.plan(ctx, &params)
	if err != nil {
		if changepreview.IsRefusal(err) || errortypes.IsNotFoundError(err) {
			return changepreview.WarnWouldFail(toolpreview.Build(summary), err), nil
		}
		return nil, err
	}

	changes, err := draftChanges(plan)
	if err != nil {
		return nil, err
	}
	preview := toolpreview.Build(draftSummary(plan.Draft), changes...)
	if readyErr := readiness(input, plan.Draft); readyErr != nil {
		return changepreview.WarnWouldFail(preview, readyErr), nil
	}
	return preview, nil
}

func pinned(id pulid.ID, version int64) *int64 {
	if id.IsNil() {
		return nil
	}
	return changepreview.PinnedVersion(version)
}

func draftChanges(
	plan *ediconnectiondraftports.ConnectionDraftPlan,
) ([]*agentpreview.RecordChange, error) {
	changes := make([]*agentpreview.RecordChange, 0, 3+len(plan.Mappings))
	add := func(change *agentpreview.RecordChange, err error) error {
		if err != nil {
			return err
		}
		changes = append(changes, change)
		return nil
	}

	partner := plan.Partner
	if partner.Change.Writes() {
		record := toolpreview.Record{
			Resource: permission.ResourceEDI,
			ID:       partner.After.ID,
			Label:    "EDI partner " + partner.After.Name,
			Version:  pinned(partner.After.ID, partner.After.Version),
		}
		if err := add(
			writeChange(record, partner.Before, partner.After, draftPartnerFields),
		); err != nil {
			return nil, err
		}
	}
	if transport := plan.Transport; transport != nil && transport.Change.Writes() {
		record := toolpreview.Record{
			Resource: permission.ResourceEDI,
			ID:       transport.After.ID,
			Label:    "Communication profile " + transport.After.Name,
			Version:  pinned(transport.After.ID, transport.After.Version),
		}
		if err := add(
			writeChange(record, transport.Before, transport.After, draftTransportFields),
		); err != nil {
			return nil, err
		}
	}
	if project := plan.Project; project != nil && project.Change.Writes() {
		record := toolpreview.Record{
			Resource: permission.ResourceEDI,
			Label:    "Lading project draft",
		}
		if project.Project != nil {
			record.ID = project.Project.ID
		}
		after := projectView(project.Files, project.Samples)
		var before *ladingProjectView
		if project.Before != nil {
			before = projectView(project.Before.Files, nil)
			for _, sample := range project.Before.Samples {
				before.Samples = append(before.Samples, sample.Path)
			}
			after.Samples = append(slices.Clone(before.Samples), after.Samples...)
		}
		if err := add(writeChange(record, before, after, draftProjectFields)); err != nil {
			return nil, err
		}
	}
	for index := range plan.ControlNumbers {
		control := &plan.ControlNumbers[index]
		record := toolpreview.Record{
			Resource: permission.ResourceEDI,
			Label:    fmt.Sprintf("%s control numbers", control.After.Kind),
		}
		if err := add(
			writeChange(record, control.Before, control.After, draftControlNumberFields),
		); err != nil {
			return nil, err
		}
	}
	for index := range plan.Mappings {
		mapping := &plan.Mappings[index]
		record := toolpreview.Record{
			Resource: permission.ResourceEDI,
			Label: fmt.Sprintf(
				"%s mapping %s",
				mapping.After.EntityType,
				mapping.After.SourceID,
			),
		}
		if err := add(
			writeChange(record, mapping.Before, mapping.After, draftMappingFields),
		); err != nil {
			return nil, err
		}
	}
	return changes, nil
}

// ladingProjectView is what a draft writes into a Lading project: its files by path, and the
// paths of its samples.
type ladingProjectView struct {
	Files   map[string]string `json:"files"`
	Samples []string          `json:"samples"`
}

func projectView(files []lading.File, samples []lading.Sample) *ladingProjectView {
	view := &ladingProjectView{Files: make(map[string]string, len(files))}
	for _, file := range files {
		view.Files[file.Path] = file.Source
	}
	for _, sample := range samples {
		view.Samples = append(view.Samples, sample.Path)
	}
	return view
}

func writeChange[T any](
	record toolpreview.Record,
	before, after *T,
	fields []string,
) (*agentpreview.RecordChange, error) {
	if before == nil {
		return toolpreview.Create(record, after, toolpreview.Only(fields...))
	}
	return toolpreview.Changed(record, before, after, toolpreview.Only(fields...))
}

func draftPartnerName(draft *ediconnectiondraftports.ConnectionDraft) string {
	for index := range draft.Partner.Fields {
		if draft.Partner.Fields[index].Field == "partner.name" &&
			draft.Partner.Fields[index].Value != "" {
			return draft.Partner.Fields[index].Value
		}
	}
	return draft.FileName
}

func draftSummary(draft *ediconnectiondraftports.ConnectionDraft) string {
	name := draftPartnerName(draft)
	sets := make([]string, 0, len(draft.Connection.Sets))
	for _, set := range draft.Connection.Sets {
		sets = append(
			sets,
			fmt.Sprintf("%s %s", set.TransactionSet, strings.ToLower(string(set.Direction))),
		)
	}
	verb := "Would set up"
	if draft.Partner.Match == ediconnectiondraftports.PartnerMatchExisting {
		verb = "Would update"
	}
	summary := fmt.Sprintf(
		"%s the EDI connection with %s from its guide %s (%d pages, %s): %d %s.",
		verb, name, draft.FileName, draft.PageCount, draft.DocumentDigest, draft.Changes,
		plural(draft.Changes, "change", "changes"),
	)
	if len(sets) > 0 {
		summary += " It exchanges " + strings.Join(sets, ", ") + "."
	}
	if draft.Partner.Match == ediconnectiondraftports.PartnerMatchNew {
		summary += " Everything new is inactive until a person enters its credentials and turns it on."
	}
	return summary + outstanding(draft)
}

func outstanding(draft *ediconnectiondraftports.ConnectionDraft) string {
	notes := make([]string, 0, 3)
	if missing := draft.CredentialsToEnter(); len(missing) > 0 {
		notes = append(notes, "Before turning it on, enter: "+strings.Join(missing, ", ")+".")
	}
	unmet := 0
	for index := range draft.Sets {
		unmet += len(draft.Sets[index].Unmet)
	}
	if unmet > 0 {
		notes = append(notes, fmt.Sprintf(
			"The drafted maps leave %d of the guide's mandatory elements unwritten, to finish in the Lading studio.",
			unmet,
		))
	}
	if draft.Project != nil && draft.Project.Errors > 0 {
		notes = append(notes, fmt.Sprintf(
			"The drafted Lading project has %d %s to fix before it can be certified.",
			draft.Project.Errors, plural(draft.Project.Errors, "error", "errors")))
	}
	if len(draft.Gaps) > 0 {
		notes = append(notes, fmt.Sprintf(
			"The guide asks for %d %s Trenova does not do or leaves unclear.",
			len(draft.Gaps), plural(len(draft.Gaps), "thing", "things")))
	}
	if len(notes) == 0 {
		return ""
	}
	return " " + strings.Join(notes, " ")
}

func plural(count int, one, many string) string {
	if count == 1 {
		return one
	}
	return many
}
