package sidebarpreference

import (
	"fmt"

	"github.com/emoss08/trenova/pkg/errortypes"
)

type SectionPreference struct {
	Key    string `json:"key"`
	Hidden bool   `json:"hidden"`
}

type ActivityPreference struct {
	PageSize    int  `json:"pageSize"`
	DefaultOpen bool `json:"defaultOpen"`
}

type Document struct {
	SchemaVersion    int                 `json:"schemaVersion"`
	Sections         []SectionPreference `json:"sections"`
	AttentionMetrics []string            `json:"attentionMetrics"`
	QuickActionIDs   []string            `json:"quickActionIds"`
	Activity         ActivityPreference  `json:"activity"`
}

func (d *Document) Validate(multiErr *errortypes.MultiError) {
	if d.SchemaVersion != DocumentSchemaVersion {
		multiErr.Add(
			"schemaVersion",
			errortypes.ErrInvalid,
			fmt.Sprintf("Schema version must be %d", DocumentSchemaVersion),
		)
	}

	d.validateSections(multiErr)
	d.validateAttentionMetrics(multiErr)
	d.validateQuickActions(multiErr)
	d.validateActivity(multiErr)
}

func (d *Document) validateSections(multiErr *errortypes.MultiError) {
	seen := make(map[string]struct{}, len(d.Sections))
	for idx, section := range d.Sections {
		keyField := fmt.Sprintf("sections[%d].key", idx)

		definition, ok := sectionDefinition(section.Key)
		if !ok {
			multiErr.Add(
				keyField,
				errortypes.ErrInvalid,
				fmt.Sprintf("Unknown sidebar section: %s", section.Key),
			)
			continue
		}

		if _, dup := seen[section.Key]; dup {
			multiErr.Add(
				keyField,
				errortypes.ErrDuplicate,
				fmt.Sprintf("Duplicate sidebar section: %s", section.Key),
			)
			continue
		}
		seen[section.Key] = struct{}{}

		if section.Hidden && !definition.Hideable {
			multiErr.Add(
				fmt.Sprintf("sections[%d].hidden", idx),
				errortypes.ErrInvalid,
				fmt.Sprintf("The %s section cannot be hidden", definition.Label),
			)
		}
	}
}

func (d *Document) validateAttentionMetrics(multiErr *errortypes.MultiError) {
	seen := make(map[string]struct{}, len(d.AttentionMetrics))
	for idx, key := range d.AttentionMetrics {
		field := fmt.Sprintf("attentionMetrics[%d]", idx)

		if !attentionMetricExists(key) {
			multiErr.Add(
				field,
				errortypes.ErrInvalid,
				fmt.Sprintf("Unknown attention metric: %s", key),
			)
			continue
		}

		if _, dup := seen[key]; dup {
			multiErr.Add(
				field,
				errortypes.ErrDuplicate,
				fmt.Sprintf("Duplicate attention metric: %s", key),
			)
			continue
		}
		seen[key] = struct{}{}
	}
}

func (d *Document) validateQuickActions(multiErr *errortypes.MultiError) {
	if len(d.QuickActionIDs) > MaxQuickActions {
		multiErr.Add(
			"quickActionIds",
			errortypes.ErrInvalid,
			fmt.Sprintf("At most %d quick actions can be selected", MaxQuickActions),
		)
	}

	seen := make(map[string]struct{}, len(d.QuickActionIDs))
	for idx, id := range d.QuickActionIDs {
		field := fmt.Sprintf("quickActionIds[%d]", idx)

		if !quickActionExists(id) {
			multiErr.Add(
				field,
				errortypes.ErrInvalid,
				fmt.Sprintf("Unknown quick action: %s", id),
			)
			continue
		}

		if _, dup := seen[id]; dup {
			multiErr.Add(
				field,
				errortypes.ErrDuplicate,
				fmt.Sprintf("Duplicate quick action: %s", id),
			)
			continue
		}
		seen[id] = struct{}{}
	}
}

func (d *Document) validateActivity(multiErr *errortypes.MultiError) {
	if !activityPageSizeAllowed(d.Activity.PageSize) {
		multiErr.Add(
			"activity.pageSize",
			errortypes.ErrInvalid,
			fmt.Sprintf("Activity page size must be one of %v", ActivityPageSizes()),
		)
	}
}

func (d *Document) Normalize() *Document {
	catalog := SectionCatalog()
	sections := make([]SectionPreference, 0, len(catalog))
	seenSections := make(map[string]struct{}, len(catalog))

	for _, section := range d.Sections {
		definition, ok := sectionDefinition(section.Key)
		if !ok {
			continue
		}
		if _, dup := seenSections[section.Key]; dup {
			continue
		}
		seenSections[section.Key] = struct{}{}

		if !definition.Hideable {
			section.Hidden = false
		}
		sections = append(sections, section)
	}

	for _, definition := range catalog {
		if _, ok := seenSections[definition.Key]; !ok {
			sections = append(sections, SectionPreference{Key: definition.Key})
		}
	}

	metrics := make([]string, 0, len(d.AttentionMetrics))
	seenMetrics := make(map[string]struct{}, len(d.AttentionMetrics))
	for _, key := range d.AttentionMetrics {
		if !attentionMetricExists(key) {
			continue
		}
		if _, dup := seenMetrics[key]; dup {
			continue
		}
		seenMetrics[key] = struct{}{}
		metrics = append(metrics, key)
	}

	actions := make([]string, 0, min(len(d.QuickActionIDs), MaxQuickActions))
	seenActions := make(map[string]struct{}, len(d.QuickActionIDs))
	for _, id := range d.QuickActionIDs {
		if len(actions) == MaxQuickActions {
			break
		}
		if !quickActionExists(id) {
			continue
		}
		if _, dup := seenActions[id]; dup {
			continue
		}
		seenActions[id] = struct{}{}
		actions = append(actions, id)
	}

	activity := d.Activity
	if !activityPageSizeAllowed(activity.PageSize) {
		activity.PageSize = DefaultActivityPageSize
	}

	return &Document{
		SchemaVersion:    DocumentSchemaVersion,
		Sections:         sections,
		AttentionMetrics: metrics,
		QuickActionIDs:   actions,
		Activity:         activity,
	}
}
