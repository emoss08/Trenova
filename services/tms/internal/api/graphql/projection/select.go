package projection

type FieldRequested func(path string) bool

type TypeSpec struct {
	TypeName      string
	FieldMap      map[string]string
	AlwaysColumns []string
	Fields        []FieldSpec
}

type FieldSpec struct {
	Name        string
	FieldMapKey string
	Special     string
	Relation    *RelationSpec
}

type RelationSpec struct {
	Target *TypeSpec
	Gate   string
}

type SelectOptions struct {
	PathPrefix string
	Gates      map[string]bool
}

type Selection struct {
	Columns   []string
	Relations map[string]RelationSelection
	Specials  map[string]bool
}

type RelationSelection struct {
	Selection Selection
}

func (s Selection) HasRelation(name string) bool {
	_, ok := s.Relations[name]
	return ok
}

func (s Selection) RelationColumns(name string) []string {
	relation, ok := s.Relations[name]
	if !ok {
		return nil
	}

	return relation.Selection.Columns
}

func (s Selection) HasSpecial(name string) bool {
	return s.Specials[name]
}

func Select(spec TypeSpec, fieldRequested FieldRequested, opts SelectOptions) Selection {
	selection := newSelection(len(spec.AlwaysColumns), len(spec.Fields))
	selector := selector{
		fieldRequested: fieldRequested,
		gates:          opts.Gates,
	}
	selector.selectSpec(&spec, opts.PathPrefix, &selection)

	return selection
}

type selector struct {
	fieldRequested FieldRequested
	gates          map[string]bool
}

func (s selector) selectSpec(spec *TypeSpec, pathPrefix string, selection *Selection) {
	seenColumns := make(map[string]struct{}, len(spec.AlwaysColumns)+len(spec.Fields))
	for _, column := range spec.AlwaysColumns {
		appendColumn(selection, seenColumns, column)
	}

	if s.fieldRequested == nil {
		return
	}

	for _, field := range spec.Fields {
		path := fieldPath(pathPrefix, field.Name)
		if !s.fieldRequested(path) {
			continue
		}

		if field.Relation != nil {
			if !s.gateAllowed(field.Relation.Gate) {
				continue
			}
			if field.FieldMapKey != "" {
				appendColumn(selection, seenColumns, spec.FieldMap[field.FieldMapKey])
			}
			child := newSelection(
				len(field.Relation.Target.AlwaysColumns),
				len(field.Relation.Target.Fields),
			)
			s.selectSpec(field.Relation.Target, path, &child)
			selection.Relations[field.Name] = RelationSelection{
				Selection: child,
			}
			continue
		}
		if field.FieldMapKey != "" {
			appendColumn(selection, seenColumns, spec.FieldMap[field.FieldMapKey])
		}
		if field.Special != "" {
			selection.Specials[field.Special] = true
		}
	}
}

func (s selector) gateAllowed(name string) bool {
	if name == "" {
		return true
	}

	allowed, ok := s.gates[name]
	return !ok || allowed
}

func newSelection(columnCap, fieldCap int) Selection {
	return Selection{
		Columns:   make([]string, 0, columnCap+fieldCap),
		Relations: make(map[string]RelationSelection),
		Specials:  make(map[string]bool),
	}
}

func appendColumn(selection *Selection, seen map[string]struct{}, column string) {
	if column == "" {
		return
	}
	if _, ok := seen[column]; ok {
		return
	}

	seen[column] = struct{}{}
	selection.Columns = append(selection.Columns, column)
}

func fieldPath(prefix, field string) string {
	if prefix == "" {
		return field
	}

	return prefix + "." + field
}
