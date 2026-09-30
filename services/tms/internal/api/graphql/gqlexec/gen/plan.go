package main

import (
	"fmt"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/99designs/gqlgen/codegen"
	"github.com/99designs/gqlgen/codegen/config"
	"github.com/99designs/gqlgen/codegen/templates"
	"github.com/vektah/gqlparser/v2/ast"
)

const shardSuffix = "exec"

type shardData struct {
	Name          string
	Package       string
	ImportPath    string
	Dir           string
	Runtime       string
	Objects       []*codegen.Object
	FieldSets     []*fieldSet
	Abstracts     []*codegen.Interface
	Inputs        []*codegen.Object
	ForeignInputs []*codegen.Object
	Marshals      []*config.TypeReference
	Unmarshals    []*config.TypeReference
	Enums         []*config.TypeReference
	ArgFields     []*codegen.Field
	Resolvers     []*resolverIface
	ChildErrs     []*childErr

	fieldSetByObj map[string]*fieldSet
	resolverByObj map[string]*resolverIface
	childErrByMsg map[string]*childErr
	marshals      map[string]*config.TypeReference
	unmarshals    map[string]*config.TypeReference
	enums         map[string]*config.TypeReference
	inputsNeeded  map[string]struct{}
	inputsLocal   map[string]struct{}
}

type fieldSet struct {
	Object *codegen.Object
	Fields []*fieldData
}

type fieldData struct {
	F         *codegen.Field
	Iface     string
	Root      string
	HasChild  bool
	ChildType string
	ChildErr  string
}

type resolverIface struct {
	Name   string
	Root   string
	Object *codegen.Object
	Fields []*codegen.Field
}

type childErr struct {
	Var string
	Msg string
}

type rootData struct {
	Data        *codegen.Data
	Runtime     string
	WorkerLimit int64
	Shards      []*shardData
	Roots       []string
}

type planner struct {
	data    *codegen.Data
	runtime string
	baseDir string
	baseImp string
	shards  map[string]*shardData
	inputs  map[string]*codegen.Object
}

func buildPlan(data *codegen.Data, runtimeImport string) (*rootData, error) {
	p := &planner{
		data:    data,
		runtime: runtimeImport,
		baseDir: filepath.Join(data.Config.Exec.Dir(), shardSuffix),
		baseImp: path.Join(data.Config.Exec.ImportPath(), shardSuffix),
		shards:  make(map[string]*shardData),
		inputs:  make(map[string]*codegen.Object),
	}

	for _, obj := range data.Objects {
		p.addObject(obj)
	}
	for _, iface := range data.Interfaces {
		s := p.shard(iface.Position)
		s.Abstracts = append(s.Abstracts, iface)
	}
	for _, in := range data.Inputs {
		p.inputs[in.Name] = in
		if in.HasUnmarshal() {
			continue
		}
		p.addInput(in)
	}

	if err := p.resolveForeignInputs(); err != nil {
		return nil, err
	}

	root := &rootData{
		Data:        data,
		Runtime:     runtimeImport,
		WorkerLimit: int64(data.Config.Exec.WorkerLimit),
		Roots:       resolverRoots(data),
	}
	for _, s := range p.shards {
		s.finalize()
		root.Shards = append(root.Shards, s)
	}
	sort.Slice(root.Shards, func(i, j int) bool { return root.Shards[i].Name < root.Shards[j].Name })

	return root, nil
}

func resolverRoots(data *codegen.Data) []string {
	var roots []string
	for _, obj := range data.Objects {
		if obj.HasResolvers() {
			roots = append(roots, templates.UcFirst(obj.Name))
		}
	}
	for _, in := range data.Inputs {
		if in.HasResolvers() {
			roots = append(roots, templates.UcFirst(in.Name))
		}
	}
	return roots
}

func (p *planner) addObject(obj *codegen.Object) {
	home := p.shard(obj.Position)
	home.Objects = append(home.Objects, obj)

	for _, f := range obj.Fields {
		s := home
		if f.FieldDefinition != nil && f.Position != nil {
			s = p.shard(f.Position)
		}
		s.addField(obj, f)
	}
}

func (p *planner) addInput(in *codegen.Object) {
	s := p.shard(in.Position)
	s.Inputs = append(s.Inputs, in)
	s.inputsLocal[in.Name] = struct{}{}
	for _, f := range in.Fields {
		s.addUnmarshal(f.TypeReference)
		if f.IsResolver {
			s.resolver(in).Fields = append(s.resolver(in).Fields, f)
		}
	}
}

func (p *planner) resolveForeignInputs() error {
	for _, s := range p.shards {
		names := make([]string, 0, len(s.inputsNeeded))
		for name := range s.inputsNeeded {
			if _, ok := s.inputsLocal[name]; !ok {
				names = append(names, name)
			}
		}
		sort.Strings(names)
		for _, name := range names {
			in, ok := p.inputs[name]
			if !ok {
				return fmt.Errorf("shard %s needs input %s, which has no unmarshaler", s.Name, name)
			}
			s.ForeignInputs = append(s.ForeignInputs, in)
		}
	}
	return nil
}

func (p *planner) shard(pos *ast.Position) *shardData {
	name := "prelude"
	if pos != nil && pos.Src != nil {
		name = shardName(pos.Src.Name)
	}

	if s, ok := p.shards[name]; ok {
		return s
	}

	pkg := name + shardSuffix
	s := &shardData{
		Name:          name,
		Package:       pkg,
		ImportPath:    path.Join(p.baseImp, pkg),
		Dir:           filepath.Join(p.baseDir, pkg),
		Runtime:       p.runtime,
		fieldSetByObj: make(map[string]*fieldSet),
		resolverByObj: make(map[string]*resolverIface),
		childErrByMsg: make(map[string]*childErr),
		marshals:      make(map[string]*config.TypeReference),
		unmarshals:    make(map[string]*config.TypeReference),
		enums:         make(map[string]*config.TypeReference),
		inputsNeeded:  make(map[string]struct{}),
		inputsLocal:   make(map[string]struct{}),
	}
	p.shards[name] = s
	return s
}

func shardName(source string) string {
	base := strings.TrimSuffix(filepath.Base(source), filepath.Ext(source))
	var b strings.Builder
	for _, r := range strings.ToLower(base) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 || unicode.IsDigit(rune(b.String()[0])) {
		return "schema" + b.String()
	}
	return b.String()
}

func (s *shardData) addField(obj *codegen.Object, f *codegen.Field) {
	set, ok := s.fieldSetByObj[obj.Name]
	if !ok {
		set = &fieldSet{Object: obj}
		s.fieldSetByObj[obj.Name] = set
		s.FieldSets = append(s.FieldSets, set)
	}

	fd := &fieldData{F: f}
	def := f.TypeReference.Definition
	switch {
	case len(def.Fields) == 0:
		fd.ChildErr = s.childErr(
			fmt.Sprintf("field of type %s does not have child fields", def.Name),
		)
	case def.Kind != ast.Object:
		fd.HasChild = true
		fd.ChildErr = s.childErr(
			fmt.Sprintf("FieldContext.Child cannot be called on type %s", def.Kind),
		)
	default:
		fd.HasChild = true
		fd.ChildType = f.ChildFieldContextTypeName()
	}

	if f.IsResolver {
		r := s.resolver(obj)
		r.Fields = append(r.Fields, f)
		fd.Iface = r.Name
		fd.Root = r.Root
	}

	s.addMarshal(f.TypeReference)
	if len(f.Args) > 0 {
		s.ArgFields = append(s.ArgFields, f)
		for _, arg := range f.Args {
			s.addUnmarshal(arg.TypeReference)
		}
	}

	set.Fields = append(set.Fields, fd)
}

func (s *shardData) resolver(obj *codegen.Object) *resolverIface {
	if r, ok := s.resolverByObj[obj.Name]; ok {
		return r
	}
	r := &resolverIface{
		Name:   "resolver" + templates.UcFirst(obj.Name),
		Root:   templates.UcFirst(obj.Name),
		Object: obj,
	}
	s.resolverByObj[obj.Name] = r
	s.Resolvers = append(s.Resolvers, r)
	return r
}

func (s *shardData) childErr(msg string) string {
	if e, ok := s.childErrByMsg[msg]; ok {
		return e.Var
	}
	e := &childErr{Var: fmt.Sprintf("errNoChild%d", len(s.childErrByMsg)), Msg: msg}
	s.childErrByMsg[msg] = e
	s.ChildErrs = append(s.ChildErrs, e)
	return e.Var
}

func (s *shardData) addEnum(t *config.TypeReference) {
	if !t.HasEnumValues() || t.IsSlice() {
		return
	}
	s.enums[enumKey(t)] = t
}

func (s *shardData) addMarshal(t *config.TypeReference) {
	name := t.MarshalFunc()
	if name == "" {
		return
	}
	if _, ok := s.marshals[name]; ok {
		return
	}
	s.marshals[name] = t
	s.addEnum(t)

	switch {
	case t.IsPtrToSlice(), t.IsPtrToIntf(), t.IsSlice():
		s.addMarshal(t.Elem())
	case t.IsPtrToPtr() && t.Unmarshaler == nil && !t.IsMarshaler:
		s.addMarshal(t.Elem())
	}
}

func (s *shardData) addUnmarshal(t *config.TypeReference) {
	name := t.UnmarshalFunc()
	if name == "" {
		return
	}
	if _, ok := s.unmarshals[name]; ok {
		return
	}
	s.unmarshals[name] = t
	s.addEnum(t)

	switch {
	case t.IsPtrToSlice(), t.IsPtrToIntf(), t.IsSlice():
		s.addUnmarshal(t.Elem())
	case t.IsPtrToPtr() && t.Unmarshaler == nil && !t.IsMarshaler:
		s.addUnmarshal(t.Elem())
	case t.Unmarshaler == nil && !t.IsMarshaler:
		s.inputsNeeded[t.GQL.Name()] = struct{}{}
	}
}

func (s *shardData) finalize() {
	sort.Slice(s.Objects, func(i, j int) bool { return s.Objects[i].Name < s.Objects[j].Name })
	sort.Slice(s.FieldSets, func(i, j int) bool {
		return s.FieldSets[i].Object.Name < s.FieldSets[j].Object.Name
	})
	sort.Slice(s.Abstracts, func(i, j int) bool { return s.Abstracts[i].Name < s.Abstracts[j].Name })
	sort.Slice(s.Inputs, func(i, j int) bool { return s.Inputs[i].Name < s.Inputs[j].Name })
	sort.Slice(s.Resolvers, func(i, j int) bool { return s.Resolvers[i].Name < s.Resolvers[j].Name })
	s.Marshals = sortedRefs(s.marshals)
	s.Unmarshals = sortedRefs(s.unmarshals)
	s.Enums = sortedRefs(s.enums)
}

func sortedRefs(m map[string]*config.TypeReference) []*config.TypeReference {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	refs := make([]*config.TypeReference, 0, len(keys))
	for _, k := range keys {
		refs = append(refs, m[k])
	}
	return refs
}

func enumKey(t *config.TypeReference) string {
	if t.IsNilable() {
		return t.Elem().UniquenessKey()
	}
	return t.UniquenessKey()
}
