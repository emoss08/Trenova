package editools

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/errortypes"
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
	draftProfileFields   = []string{
		fieldName, fieldStatus, "transactionSet", "direction", "templateId", "x12VersionOverride",
		"functionalGroupId", "envelope", "acknowledgment", "validationMode", "partnerSettings",
	}
	draftMappingFields = []string{
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
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	if err := common.GuardPreview(t, &params); err != nil {
		return nil, err
	}

	summary := "Would set up an EDI connection from the partner's guide."
	input, plan, err := t.plan(ctx, &params)
	if err != nil {
		if common.IsRefusal(err) || errortypes.IsNotFoundError(err) {
			return common.WarnWouldFail(toolpreview.Build(summary), err), nil
		}
		return nil, err
	}

	changes, err := draftChanges(plan)
	if err != nil {
		return nil, err
	}
	preview := toolpreview.Build(draftSummary(plan.Draft), changes...)
	if readyErr := readiness(input, plan.Draft); readyErr != nil {
		return common.WarnWouldFail(preview, readyErr), nil
	}
	return preview, nil
}

func pinned(id pulid.ID, version int64) *int64 {
	if id.IsNil() {
		return nil
	}
	return common.PinnedVersion(version)
}

func draftChanges(plan *ediports.ConnectionDraftPlan) ([]*agent.RecordChange, error) {
	changes := make([]*agent.RecordChange, 0, 2+len(plan.DocumentProfiles)+len(plan.Mappings))
	add := func(change *agent.RecordChange, err error) error {
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
	for index := range plan.DocumentProfiles {
		profile := &plan.DocumentProfiles[index]
		if !profile.Change.Writes() {
			continue
		}
		record := toolpreview.Record{
			Resource: permission.ResourceEDI,
			ID:       profile.After.ID,
			Label:    "Document profile " + profile.After.Name,
			Version:  pinned(profile.After.ID, profile.After.Version),
		}
		if err := add(
			writeChange(record, profile.Before, profile.After, draftProfileFields),
		); err != nil {
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

func writeChange[T any](
	record toolpreview.Record,
	before, after *T,
	fields []string,
) (*agent.RecordChange, error) {
	if before == nil {
		return toolpreview.Create(record, after, toolpreview.Only(fields...))
	}
	return toolpreview.Changed(record, before, after, toolpreview.Only(fields...))
}

func draftSummary(draft *ediports.ConnectionDraft) string {
	name := draft.FileName
	for index := range draft.Partner.Fields {
		if draft.Partner.Fields[index].Field == "partner.name" &&
			draft.Partner.Fields[index].Value != "" {
			name = draft.Partner.Fields[index].Value
		}
	}
	sets := make([]string, 0, len(draft.Connection.Sets))
	for _, set := range draft.Connection.Sets {
		sets = append(
			sets,
			fmt.Sprintf("%s %s", set.TransactionSet, strings.ToLower(string(set.Direction))),
		)
	}
	verb := "Would set up"
	if draft.Partner.Match == ediports.PartnerMatchExisting {
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
	if draft.Partner.Match == ediports.PartnerMatchNew {
		summary += " Everything new is inactive until a person enters its credentials and turns it on."
	}
	return summary + outstanding(draft)
}

func outstanding(draft *ediports.ConnectionDraft) string {
	notes := make([]string, 0, 3)
	if transport := draft.Transport; transport != nil {
		missing := make([]string, 0, len(transport.Credentials))
		for _, credential := range transport.Credentials {
			if credential.Missing && credential.Required {
				missing = append(missing, credential.Label)
			}
		}
		if len(missing) > 0 {
			notes = append(notes, "Before turning it on, enter: "+strings.Join(missing, ", ")+".")
		}
	}
	unmet := 0
	for index := range draft.DocumentProfiles {
		unmet += len(draft.DocumentProfiles[index].Template.Unmet)
	}
	if unmet > 0 {
		notes = append(notes, fmt.Sprintf(
			"The chosen templates leave %d of the guide's segment rules unmet for the template designer.",
			unmet,
		))
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
