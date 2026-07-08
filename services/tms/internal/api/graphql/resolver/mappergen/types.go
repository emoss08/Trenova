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
	Skip     bool              `yaml:"skip"`
	Exclude  []string          `yaml:"exclude"`
	Aliases  map[string]string `yaml:"aliases"`
	Imports  map[string]string `yaml:"imports"`
	Defaults map[string]string `yaml:"defaults"`
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
	InputName string
	Fields    []patchAssignment
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
	Body      []string
}

type parsedPackage struct {
	Name    string
	Files   []*ast.File
	Structs map[string]goStruct
}
