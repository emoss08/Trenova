package formulatemplate

import (
	"maps"
	"slices"

	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
)

type Seed struct {
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID

	Name        string
	Description string
	Source      string
	Inputs      []tarifftypes.InputDecl
	Imports     []tarifftypes.Import
	TestCount   int32
	Metadata    map[string]any

	Status Status

	SourceTemplateID    *pulid.ID
	SourceVersionNumber *int64
}

func (s Seed) Build() *FormulaTemplate {
	status := s.Status
	if status == "" {
		status = StatusDraft
	}
	inputs := slices.Clone(s.Inputs)
	if inputs == nil {
		inputs = []tarifftypes.InputDecl{}
	}
	imports := slices.Clone(s.Imports)
	if imports == nil {
		imports = []tarifftypes.Import{}
	}
	var metadata map[string]any
	if s.Metadata != nil {
		metadata = maps.Clone(s.Metadata)
	}

	return &FormulaTemplate{
		OrganizationID:       s.OrganizationID,
		BusinessUnitID:       s.BusinessUnitID,
		Name:                 s.Name,
		Description:          s.Description,
		Source:               s.Source,
		Inputs:               inputs,
		Imports:              imports,
		TestCount:            s.TestCount,
		Status:               status,
		Metadata:             metadata,
		SourceTemplateID:     s.SourceTemplateID,
		SourceVersionNumber:  s.SourceVersionNumber,
		CurrentVersionNumber: 1,
	}
}

func SeedFromTemplate(source *FormulaTemplate) Seed {
	sourceID := source.ID
	sourceVersion := source.CurrentVersionNumber

	return Seed{
		OrganizationID:      source.OrganizationID,
		BusinessUnitID:      source.BusinessUnitID,
		Name:                source.Name,
		Description:         source.Description,
		Source:              source.Source,
		Inputs:              source.Inputs,
		Imports:             source.Imports,
		TestCount:           source.TestCount,
		Metadata:            source.Metadata,
		SourceTemplateID:    &sourceID,
		SourceVersionNumber: &sourceVersion,
	}
}

func SeedFromVersion(snapshot *FormulaTemplateVersion) Seed {
	sourceID := snapshot.TemplateID
	sourceVersion := snapshot.VersionNumber

	return Seed{
		OrganizationID:      snapshot.OrganizationID,
		BusinessUnitID:      snapshot.BusinessUnitID,
		Name:                snapshot.Name,
		Description:         snapshot.Description,
		Source:              snapshot.Source,
		Inputs:              snapshot.Inputs,
		Imports:             snapshot.Imports,
		TestCount:           snapshot.TestCount,
		Metadata:            snapshot.Metadata,
		SourceTemplateID:    &sourceID,
		SourceVersionNumber: &sourceVersion,
	}
}
