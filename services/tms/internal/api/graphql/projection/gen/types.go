package main

import "github.com/vektah/gqlparser/v2/ast"

type manifest struct {
	Aliases        map[string]map[string]string   `yaml:"aliases"`
	Virtuals       map[string][]string            `yaml:"virtuals"`
	Specials       map[string]map[string][]string `yaml:"specials"`
	Gates          map[string]map[string]string   `yaml:"gates"`
	Always         map[string][]string            `yaml:"always"`
	ModelOverrides map[string]string              `yaml:"modelOverrides"`
}

type gqlgenConfig struct {
	Models map[string]gqlgenModel `yaml:"models"`
}

type gqlgenModel struct {
	Model []string `yaml:"model"`
}

type generatorOptions struct {
	ManifestPath string
	SchemaDir    string
	OutputPath   string
	GqlgenPath   string
	DomainDir    string
	GoModPath    string
}

type tableColumnMap struct {
	Values map[string]string
}

type goStruct struct {
	PackagePath string
	PackageName string
	Name        string
	FullName    string
	IsEntity    bool
	Fields      map[string]goField
}

type goField struct {
	JSONName      string
	GoName        string
	TypeName      string
	ColumnName    string
	IsColumn      bool
	IsRelation    bool
	RelationKind  string
	RelationLocal string
}

type discovery struct {
	Schema     *ast.Schema
	Manifest   manifest
	Gqlgen     gqlgenConfig
	Structs    map[string]goStruct
	Selections map[string]typeSelection
	Skipped    map[string]string
}

type typeSelection struct {
	TypeName string
	Struct   goStruct
	Columns  tableColumnMap
}

type generatedSpec struct {
	Name          string
	AlwaysColumns []string
	Fields        []generatedField
}

type generatedField struct {
	Name     string
	Column   string
	Special  string
	Relation *generatedRelation
}

type generatedRelation struct {
	Target string
	Gate   string
}
