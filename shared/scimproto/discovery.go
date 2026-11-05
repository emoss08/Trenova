package scimproto

type Supported struct {
	Supported bool `json:"supported"`
}

type BulkSupport struct {
	Supported      bool `json:"supported"`
	MaxOperations  int  `json:"maxOperations"`
	MaxPayloadSize int  `json:"maxPayloadSize"`
}

type FilterSupport struct {
	Supported  bool `json:"supported"`
	MaxResults int  `json:"maxResults"`
}

type AuthenticationScheme struct {
	Type        string `json:"type"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Primary     bool   `json:"primary"`
}

type ServiceProviderConfig struct {
	Schemas               []string               `json:"schemas"`
	Patch                 Supported              `json:"patch"`
	Bulk                  BulkSupport            `json:"bulk"`
	Filter                FilterSupport          `json:"filter"`
	ChangePassword        Supported              `json:"changePassword"`
	Sort                  Supported              `json:"sort"`
	ETag                  Supported              `json:"etag"`
	AuthenticationSchemes []AuthenticationScheme `json:"authenticationSchemes"`
	Meta                  *Meta                  `json:"meta"`
}

func NewServiceProviderConfig(baseURL string, maxResults int) *ServiceProviderConfig {
	return &ServiceProviderConfig{
		Schemas: []string{SchemaServiceProviderConfig},
		Patch:   Supported{Supported: true},
		Bulk:    BulkSupport{},
		Filter:  FilterSupport{Supported: true, MaxResults: maxResults},
		AuthenticationSchemes: []AuthenticationScheme{{
			Type:        "oauthbearertoken",
			Name:        "OAuth Bearer Token",
			Description: "A SCIM token issued for one directory, sent as an HTTP bearer token",
			Primary:     true,
		}},
		Meta: &Meta{
			ResourceType: "ServiceProviderConfig",
			Location:     baseURL + "/ServiceProviderConfig",
		},
	}
}

type ResourceType struct {
	Schemas          []string `json:"schemas"`
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	Endpoint         string   `json:"endpoint"`
	Description      string   `json:"description"`
	Schema           string   `json:"schema"`
	SchemaExtensions []any    `json:"schemaExtensions"`
	Meta             *Meta    `json:"meta"`
}

func NewResourceTypes(baseURL string) []*ResourceType {
	return []*ResourceType{
		{
			Schemas:          []string{SchemaResourceType},
			ID:               ResourceTypeUser,
			Name:             ResourceTypeUser,
			Endpoint:         "/Users",
			Description:      "A person in the directory's organization",
			Schema:           SchemaUser,
			SchemaExtensions: []any{},
			Meta: &Meta{
				ResourceType: "ResourceType",
				Location:     baseURL + "/ResourceTypes/" + ResourceTypeUser,
			},
		},
		{
			Schemas:          []string{SchemaResourceType},
			ID:               ResourceTypeGroup,
			Name:             ResourceTypeGroup,
			Endpoint:         "/Groups",
			Description:      "A directory group; a group role mapping turns membership into a role",
			Schema:           SchemaGroup,
			SchemaExtensions: []any{},
			Meta: &Meta{
				ResourceType: "ResourceType",
				Location:     baseURL + "/ResourceTypes/" + ResourceTypeGroup,
			},
		},
	}
}

type SchemaAttribute struct {
	Name          string            `json:"name"`
	Type          string            `json:"type"`
	MultiValued   bool              `json:"multiValued"`
	Description   string            `json:"description,omitempty"`
	Required      bool              `json:"required"`
	CaseExact     bool              `json:"caseExact"`
	Mutability    string            `json:"mutability"`
	Returned      string            `json:"returned"`
	Uniqueness    string            `json:"uniqueness"`
	SubAttributes []SchemaAttribute `json:"subAttributes,omitempty"`
}

type Schema struct {
	Schemas     []string          `json:"schemas"`
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Attributes  []SchemaAttribute `json:"attributes"`
	Meta        *Meta             `json:"meta"`
}

func attribute(name, kind string, required bool, uniqueness string) SchemaAttribute {
	return SchemaAttribute{
		Name:       name,
		Type:       kind,
		Required:   required,
		Mutability: "readWrite",
		Returned:   "default",
		Uniqueness: uniqueness,
	}
}

func multiValuedAttribute(name string) SchemaAttribute {
	return SchemaAttribute{
		Name:        name,
		Type:        "complex",
		MultiValued: true,
		Mutability:  "readWrite",
		Returned:    "default",
		Uniqueness:  "none",
		SubAttributes: []SchemaAttribute{
			attribute("value", "string", false, "none"),
			attribute("type", "string", false, "none"),
			attribute("primary", "boolean", false, "none"),
		},
	}
}

func NewSchemas(baseURL string) []*Schema {
	name := SchemaAttribute{
		Name:       "name",
		Type:       "complex",
		Mutability: "readWrite",
		Returned:   "default",
		Uniqueness: "none",
		SubAttributes: []SchemaAttribute{
			attribute("formatted", "string", false, "none"),
			attribute("familyName", "string", false, "none"),
			attribute("givenName", "string", false, "none"),
			attribute("middleName", "string", false, "none"),
		},
	}
	members := SchemaAttribute{
		Name:        "members",
		Type:        "complex",
		MultiValued: true,
		Mutability:  "readWrite",
		Returned:    "default",
		Uniqueness:  "none",
		SubAttributes: []SchemaAttribute{
			attribute("value", "string", false, "none"),
			attribute("display", "string", false, "none"),
		},
	}

	return []*Schema{
		{
			Schemas:     []string{SchemaSchema},
			ID:          SchemaUser,
			Name:        ResourceTypeUser,
			Description: "User Account",
			Attributes: []SchemaAttribute{
				attribute("userName", "string", true, "server"),
				attribute("externalId", "string", false, "none"),
				name,
				attribute("displayName", "string", false, "none"),
				attribute("title", "string", false, "none"),
				attribute("timezone", "string", false, "none"),
				attribute("locale", "string", false, "none"),
				attribute("active", "boolean", false, "none"),
				multiValuedAttribute("emails"),
				multiValuedAttribute("phoneNumbers"),
			},
			Meta: &Meta{ResourceType: "Schema", Location: baseURL + "/Schemas/" + SchemaUser},
		},
		{
			Schemas:     []string{SchemaSchema},
			ID:          SchemaGroup,
			Name:        ResourceTypeGroup,
			Description: "Group",
			Attributes: []SchemaAttribute{
				attribute("displayName", "string", true, "server"),
				attribute("externalId", "string", false, "none"),
				members,
			},
			Meta: &Meta{ResourceType: "Schema", Location: baseURL + "/Schemas/" + SchemaGroup},
		},
	}
}
