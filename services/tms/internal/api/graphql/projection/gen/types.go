package main

type manifest struct {
	Types map[string]typeManifest `yaml:"types"`
	Skip  []string                `yaml:"skip"`
}

type typeManifest struct {
	Always    []string                    `yaml:"always"`
	Aliases   map[string]string           `yaml:"aliases"`
	Virtuals  map[string]string           `yaml:"virtuals"`
	Relations map[string]relationManifest `yaml:"relations"`
}

type relationManifest struct {
	Target    string `yaml:"target"`
	Gate      string `yaml:"gate"`
	ColumnKey string `yaml:"columnKey"`
}

type generatorOptions struct {
	ManifestPath string
	SchemaDir    string
	OutputPath   string
	FieldMaps    map[string]map[string]string
}

type generatedSpec struct {
	Name          string
	FieldMap      string
	AlwaysColumns []string
	Fields        []generatedField
}

type generatedField struct {
	Name        string
	FieldMapKey string
	Special     string
	Relation    *generatedRelation
}

type generatedRelation struct {
	Target string
	Gate   string
}
