package platformcatalog

type ProductKey string

type FeatureKey string

type MeterKey string

type PackKey string

type GraphQLSource string

type RouteAccessClass string

const (
	RouteAccessClassAccountShell RouteAccessClass = "account_shell"
	RouteAccessClassProduct      RouteAccessClass = "product"
	RouteAccessClassUnclassified RouteAccessClass = "unclassified"
)

const (
	GraphQLOperationQuery    = "Query"
	GraphQLOperationMutation = "Mutation"
)

type Product struct {
	Key         ProductKey   `json:"key"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Features    []FeatureKey `json:"features"`
}

type Feature struct {
	Key                    FeatureKey         `json:"key"`
	ProductKey             ProductKey         `json:"productKey"`
	Name                   string             `json:"name"`
	Description            string             `json:"description"`
	RequiresFeatures       []FeatureKey       `json:"requiresFeatures"`
	LegacyGrantingFeatures []FeatureKey       `json:"legacyGrantingFeatures,omitempty"`
	Routes                 []RouteRef         `json:"routes"`
	GraphQLSources         []GraphQLSource    `json:"graphqlSources,omitempty"`
	GraphQLRootFields      []GraphQLRootField `json:"graphqlRootFields,omitempty"`
	Permissions            []PermissionRef    `json:"permissions"`
	Meters                 []MeterKey         `json:"meters"`
}

type Pack struct {
	Key         PackKey      `json:"key"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Standalone  bool         `json:"standalone"`
	Features    []FeatureKey `json:"features"`
}

type GraphQLRootField struct {
	Operation string `json:"operation"`
	Field     string `json:"field"`
}

func (f GraphQLRootField) Key() string {
	return f.Operation + "." + f.Field
}

type Meter struct {
	Key         MeterKey   `json:"key"`
	ProductKey  ProductKey `json:"productKey"`
	FeatureKey  FeatureKey `json:"featureKey,omitempty"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Unit        string     `json:"unit"`
}

type RouteRef struct {
	Method string `json:"method"`
	Path   string `json:"path"`
}

type PermissionRef struct {
	Resource  string `json:"resource"`
	Operation string `json:"operation"`
}

type RoutePolicy struct {
	AccessClass RouteAccessClass `json:"accessClass"`
	FeatureKey  FeatureKey       `json:"featureKey,omitempty"`
}

type UsageLimit struct {
	MeterKey MeterKey `json:"meterKey"`
	Limit    int64    `json:"limit"`
	Window   string   `json:"window"`
}

type CatalogProvider interface {
	Products() []Product
	Features() []Feature
	Meters() []Meter
}

type PackProvider interface {
	Packs() []Pack
}
