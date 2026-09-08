package main

import "go/ast"

type generatorOptions struct {
	ManifestPath string
	OutputPath   string
	OutputDir    string
	GqlgenPath   string
	ModelPath    string
	DomainDir    string
	ResolverDir  string
	GoModPath    string
	ReportPath   string
	Verbose      bool
}

type declineStage string

const (
	declineStageType   declineStage = "type"
	declineStageCreate declineStage = "create"
	declineStagePatch  declineStage = "patch"
)

type decline struct {
	TypeName string
	Stage    declineStage
	Field    string
	Reason   string
	Routine  bool
}

type declineLog struct {
	entries []decline
}

func (l *declineLog) note(typeName string, stage declineStage, field, reason string) {
	l.entries = append(l.entries, decline{
		TypeName: typeName,
		Stage:    stage,
		Field:    field,
		Reason:   reason,
	})
}

func (l *declineLog) noteRoutine(typeName string, stage declineStage, reason string) {
	l.entries = append(l.entries, decline{
		TypeName: typeName,
		Stage:    stage,
		Reason:   reason,
		Routine:  true,
	})
}

const (
	typeNameInt    = "int"
	typeNameInt64  = "int64"
	typeNameString = "string"
)

type manifest struct {
	Types map[string]typeOverride `yaml:"types"`
}

type typeOverride struct {
	Skip      bool              `yaml:"skip"`
	Exclude   []string          `yaml:"exclude"`
	Aliases   map[string]string `yaml:"aliases"`
	Imports   map[string]string `yaml:"imports"`
	Defaults  map[string]string `yaml:"defaults"`
	Required  []string          `yaml:"required"`
	Clearable []string          `yaml:"clearable"`
}

type gqlgenConfig struct {
	Models map[string]gqlgenModel `yaml:"models"`
}

type gqlgenModel struct {
	Model []string `yaml:"model"`
}

type modelBinding struct {
	GraphQLName string
	ImportPath  string
	PackageName string
	GoName      string
}

type goStruct struct {
	PackageName string
	Name        string
	Fields      map[string]goField
}

type goField struct {
	GoName   string
	JSONName string
	Type     typeRef
	Bun      bunTag
}

type bunTag struct {
	Present  bool
	Ignored  bool
	Nullzero bool
}

type typeRef struct {
	Name          string
	Pointer       bool
	Slice         bool
	Map           bool
	Omittable     bool
	OmittableType *typeRef
}

type generatedType struct {
	Name        string
	Binding     modelBinding
	Create      *generatedCreate
	Patch       *generatedPatch
	Imports     []generatedImport
	NeedsPulID  bool
	NeedsAuth   bool
	NeedsDomain bool
}

type generatedCreate struct {
	InputName string
	Parses    []parseAssignment
	Defaults  []defaultAssignment
	Fields    []structAssignment
}

type generatedPatch struct {
	InputName       string
	Fields          []patchAssignment
	NeedsErrortypes bool
}

type parseAssignment struct {
	VarName    string
	Expression string
}

type defaultAssignment struct {
	VarName       string
	FieldName     string
	DefaultValue  string
	InputOverride string
}

type structAssignment struct {
	FieldName  string
	Expression string
}

type patchAssignment struct {
	FieldName string
	Guard     string
	ValueName string
	Required  bool
	Body      []string
}

type parsedPackage struct {
	Name      string
	Files     []*ast.File
	Structs   map[string]goStruct
	TypeNames map[string]struct{}
}
