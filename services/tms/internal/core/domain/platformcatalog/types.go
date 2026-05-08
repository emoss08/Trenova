package platformcatalog

type ProductKey string

type FeatureKey string

type MeterKey string

type Product struct {
	Key         ProductKey   `json:"key"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Features    []FeatureKey `json:"features"`
}

type Feature struct {
	Key              FeatureKey      `json:"key"`
	ProductKey       ProductKey      `json:"productKey"`
	Name             string          `json:"name"`
	Description      string          `json:"description"`
	RequiresFeatures []FeatureKey    `json:"requiresFeatures"`
	Routes           []RouteRef      `json:"routes"`
	Permissions      []PermissionRef `json:"permissions"`
	Meters           []MeterKey      `json:"meters"`
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
