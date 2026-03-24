package services

import (
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/pkg/pagination"
)

var CatalogDefinitions = []CatalogItem{
	{
		Type:          integration.TypeSamsara,
		Name:          "Samsara",
		Description:   "Seamlessly connect your Samsara account to Trenova for real-time telematics data, driver performance insights, and streamlined fleet management. Unlock the full potential of your fleet with our powerful integration.",
		Category:      integration.CategoryTelematics,
		CategoryLabel: "Telematics",
		LogoURL:       "/integrations/logos/samsara.webp",
		LogoLightURL:  "/integrations/logos/samsara.webp",
		LogoDarkURL:   "/integrations/logos/samsara_logo_white.webp",
		DocsURL:       "https://developers.samsara.com/docs/tms-integration",
		WebsiteURL:    "https://www.samsara.com/",
		Color:         "#002e8a",
		Links: []CatalogLink{
			{
				Kind:  CatalogLinkKindDocs,
				Label: "Docs",
				URL:   "https://developers.samsara.com/docs/tms-integration",
			},
			{
				Kind:  CatalogLinkKindWebsite,
				Label: "Website",
				URL:   "https://www.samsara.com/",
			},
		},
		Featured:           true,
		SortOrder:          10,
		PrimaryActionLabel: "View Integration",
	},
	{
		Type:          integration.TypeGoogleMaps,
		Name:          "Google Maps",
		Description:   "Routing and geocoding",
		Category:      integration.CategoryMappingRouting,
		CategoryLabel: "Mapping & Routing",
		LogoURL:       "/integrations/logos/googleMaps.svg",
		LogoLightURL:  "/integrations/logos/googleMaps.svg",
		LogoDarkURL:   "/integrations/logos/googleMaps.svg",
		DocsURL:       "https://developers.google.com/maps/documentation",
		WebsiteURL:    "https://maps.google.com/",
		Color:         "#8a0000",
		Links: []CatalogLink{
			{
				Kind:  CatalogLinkKindDocs,
				Label: "Docs",
				URL:   "https://developers.google.com/maps/documentation",
			},
			{
				Kind:  CatalogLinkKindWebsite,
				Label: "Website",
				URL:   "https://maps.google.com/",
			},
		},
		Featured:           false,
		SortOrder:          20,
		PrimaryActionLabel: "View Integration",
	},
}

type CatalogLinkKind string

const (
	CatalogLinkKindDocs    CatalogLinkKind = "docs"
	CatalogLinkKindWebsite CatalogLinkKind = "website"
	CatalogLinkKindSupport CatalogLinkKind = "support"
	CatalogLinkKindAPI     CatalogLinkKind = "api"
)

type CatalogConnectionStatus string

const (
	CatalogConnectionStatusConnected    CatalogConnectionStatus = "connected"
	CatalogConnectionStatusDisconnected CatalogConnectionStatus = "disconnected"
)

type CatalogConfigurationStatus string

const (
	CatalogConfigurationStatusConfigured CatalogConfigurationStatus = "configured"
	CatalogConfigurationStatusNeedsSetup CatalogConfigurationStatus = "needs_setup"
)

type CatalogLink struct {
	Kind  CatalogLinkKind `json:"kind"`
	Label string          `json:"label"`
	URL   string          `json:"url"`
}

type CatalogStatus struct {
	Connection         CatalogConnectionStatus    `json:"connection"`
	ConnectionLabel    string                     `json:"connectionLabel"`
	Configuration      CatalogConfigurationStatus `json:"configuration"`
	ConfigurationLabel string                     `json:"configurationLabel"`
}

type CatalogItem struct {
	Type                integration.Type              `json:"type"`
	Name                string                        `json:"name"`
	Description         string                        `json:"description"`
	Category            integration.Category          `json:"category"`
	CategoryLabel       string                        `json:"categoryLabel"`
	LogoURL             string                        `json:"logoUrl"`
	LogoLightURL        string                        `json:"logoLightUrl,omitempty"`
	LogoDarkURL         string                        `json:"logoDarkUrl,omitempty"`
	Color               string                        `json:"color,omitempty"`
	DocsURL             string                        `json:"docsUrl,omitempty"`
	WebsiteURL          string                        `json:"websiteUrl,omitempty"`
	Links               []CatalogLink                 `json:"links"`
	Featured            bool                          `json:"featured"`
	SortOrder           int                           `json:"sortOrder"`
	PrimaryActionLabel  string                        `json:"primaryActionLabel"`
	Enabled             bool                          `json:"enabled"`
	Configured          bool                          `json:"configured"`
	Status              CatalogStatus                 `json:"status"`
	ConfigSpec          []integration.ConfigFieldSpec `json:"configSpec"`
	SupportsTestConnect bool                          `json:"supportsTestConnect"`
}

type CatalogResponse struct {
	Items []CatalogItem `json:"items"`
}

type TestConnectionResponse struct {
	Provider  integration.Type `json:"provider"`
	Success   bool             `json:"success"`
	CheckedAt int64            `json:"checkedAt"`
}

type UpdateConfigRequest struct {
	TenantInfo    pagination.TenantInfo `json:"-"`
	Enabled       bool                  `json:"enabled"`
	Configuration map[string]string     `json:"configuration"`
}

type ConfigFieldValue struct {
	Key      string `json:"key"`
	Value    string `json:"value,omitempty"`
	HasValue bool   `json:"hasValue"`
}

type ConfigResponse struct {
	Type      integration.Type              `json:"type"`
	Enabled   bool                          `json:"enabled"`
	Fields    []ConfigFieldValue            `json:"fields"`
	Spec      []integration.ConfigFieldSpec `json:"spec"`
	UpdatedAt int64                         `json:"updatedAt"`
}
