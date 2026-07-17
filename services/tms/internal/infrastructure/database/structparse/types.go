package structparse

type Model struct {
	PackageName  string
	StructName   string
	TableName    string
	Alias        string
	Fields       []Field
	Relations    []Relation
	M2MRelations []M2MRelation
}

type Field struct {
	GoName     string
	ColumnName string
	JSONName   string
	GoType     string
	SQLType    string
	IsPK       bool
	IsScanOnly bool
	IsNotNull  bool
	IsNullZero bool
	IsArray    bool
	IsPointer  bool
}

type RelationKind string

const (
	RelationBelongsTo RelationKind = "belongs-to"
	RelationHasOne    RelationKind = "has-one"
	RelationHasMany   RelationKind = "has-many"
)

type JoinPair struct {
	Local  string
	Remote string
}

type Relation struct {
	GoName    string
	JSONName  string
	GoType    string
	Kind      RelationKind
	JoinPairs []JoinPair
}

type M2MRelation struct {
	GoName       string
	JSONName     string
	GoType       string
	ThroughTable string
	JoinSpec     string
}

type EnumDef struct {
	TypeName string
	Values   []string
}

func (m *Model) PKColumns() []string {
	var pks []string
	for _, f := range m.Fields {
		if f.IsPK {
			pks = append(pks, f.ColumnName)
		}
	}
	return pks
}

func (m *Model) InsertableColumns() []string {
	var cols []string
	for _, f := range m.Fields {
		if !f.IsScanOnly {
			cols = append(cols, f.ColumnName)
		}
	}
	return cols
}

func (m *Model) FieldMapEntries() []Field {
	var entries []Field
	for _, f := range m.Fields {
		if f.JSONName != "" {
			entries = append(entries, f)
		}
	}
	return entries
}

func (m *Model) HasTenantFields() bool {
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

func (m *Model) HasRelations() bool {
	return len(m.Relations) > 0
}

func (m *Model) FilterableFields() []Field {
	var entries []Field
	for _, f := range m.Fields {
		if f.JSONName != "" && !f.IsScanOnly {
			entries = append(entries, f)
		}
	}
	return entries
}

func (m *Model) Field(goName string) (*Field, bool) {
	for i := range m.Fields {
		if m.Fields[i].GoName == goName {
			return &m.Fields[i], true
		}
	}
	return nil, false
}

func (m *Model) Relation(goName string) (*Relation, bool) {
	for i := range m.Relations {
		if m.Relations[i].GoName == goName {
			return &m.Relations[i], true
		}
	}
	return nil, false
}
