package main

type ModelInfo struct {
	PackageName string
	StructName  string
	TableName   string
	Alias       string
	Fields      []FieldInfo
	Relations   []RelationInfo
}

type FieldInfo struct {
	GoName     string
	ColumnName string
	JSONName   string
	IsPK       bool
	IsScanOnly bool
}

type RelationInfo struct {
	GoName string
}

func (m *ModelInfo) PKColumns() []string {
	var pks []string
	for _, f := range m.Fields {
		if f.IsPK {
			pks = append(pks, f.ColumnName)
		}
	}
	return pks
}

func (m *ModelInfo) InsertableColumns() []string {
	var cols []string
	for _, f := range m.Fields {
		if !f.IsScanOnly {
			cols = append(cols, f.ColumnName)
		}
	}
	return cols
}

func (m *ModelInfo) FieldMapEntries() []FieldInfo {
	var entries []FieldInfo
	for _, f := range m.Fields {
		if f.JSONName != "" {
			entries = append(entries, f)
		}
	}
	return entries
}

func (m *ModelInfo) HasTenantFields() bool {
	hasOrg, hasBU := false, false
	for _, f := range m.Fields {
		if f.GoName == "OrganizationID" {
			hasOrg = true
		}
		if f.GoName == "BusinessUnitID" {
			hasBU = true
		}
	}
	return hasOrg && hasBU
}

func (m *ModelInfo) HasRelations() bool {
	return len(m.Relations) > 0
}

func (m *ModelInfo) FilterableFields() []FieldInfo {
	var entries []FieldInfo
	for _, f := range m.Fields {
		if f.JSONName != "" && !f.IsScanOnly {
			entries = append(entries, f)
		}
	}
	return entries
}
