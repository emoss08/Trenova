package accountingconfig

import (
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
)

type Config struct {
	QuickBooks      QuickBooksConfig      `mapstructure:"quickbooks"`
	Xero            XeroConfig            `mapstructure:"xero"`
	BusinessCentral BusinessCentralConfig `mapstructure:"businessCentral"`
	NetSuite        NetSuiteConfig        `mapstructure:"netSuite"`
}

type NetSuiteConfig struct {
	ClientID     string `mapstructure:"clientId"`
	ClientSecret string `mapstructure:"clientSecret"`
	RedirectURL  string `mapstructure:"redirectUrl"  validate:"omitempty,url"`
}

const netSuiteCallbackPath = "/admin/integrations/netsuite/callback"

func (c *NetSuiteConfig) GetRedirectURL(app *appconfig.Config) string {
	if c.RedirectURL != "" {
		return c.RedirectURL
	}
	if base := app.GetWebBaseURL(); base != "" {
		return base + netSuiteCallbackPath
	}
	return ""
}

func (c *NetSuiteConfig) IsConfigured(app *appconfig.Config) bool {
	return strings.TrimSpace(c.ClientID) != "" &&
		strings.TrimSpace(c.ClientSecret) != "" &&
		c.GetRedirectURL(app) != ""
}

type BusinessCentralConfig struct {
	ClientID       string `mapstructure:"clientId"`
	ClientSecret   string `mapstructure:"clientSecret"`
	RedirectURL    string `mapstructure:"redirectUrl"    validate:"omitempty,url"`
	WebhookBaseURL string `mapstructure:"webhookBaseUrl" validate:"omitempty,url"`
}

const (
	businessCentralCallbackPath = "/admin/integrations/business-central/callback"
	apiBasePath                 = "/api/v1"
)

func (c *BusinessCentralConfig) GetRedirectURL(app *appconfig.Config) string {
	if c.RedirectURL != "" {
		return c.RedirectURL
	}
	if base := app.GetWebBaseURL(); base != "" {
		return base + businessCentralCallbackPath
	}
	return ""
}

func (c *BusinessCentralConfig) GetWebhookBaseURL(app *appconfig.Config) string {
	if base := strings.TrimSuffix(strings.TrimSpace(c.WebhookBaseURL), "/"); base != "" {
		return base
	}
	if base := app.GetWebBaseURL(); base != "" {
		return base + apiBasePath
	}
	return ""
}

func (c *BusinessCentralConfig) IsConfigured(app *appconfig.Config) bool {
	return strings.TrimSpace(c.ClientID) != "" &&
		strings.TrimSpace(c.ClientSecret) != "" &&
		c.GetRedirectURL(app) != ""
}

type XeroConfig struct {
	ClientID     string `mapstructure:"clientId"`
	ClientSecret string `mapstructure:"clientSecret"`
	WebhookKey   string `mapstructure:"webhookKey"`
	RedirectURL  string `mapstructure:"redirectUrl"  validate:"omitempty,url"`
}

const xeroCallbackPath = "/admin/integrations/xero/callback"

func (c *XeroConfig) GetRedirectURL(app *appconfig.Config) string {
	if c.RedirectURL != "" {
		return c.RedirectURL
	}
	if base := app.GetWebBaseURL(); base != "" {
		return base + xeroCallbackPath
	}
	return ""
}

func (c *XeroConfig) IsConfigured(app *appconfig.Config) bool {
	return strings.TrimSpace(c.ClientID) != "" &&
		strings.TrimSpace(c.ClientSecret) != "" &&
		c.GetRedirectURL(app) != ""
}

type QuickBooksConfig struct {
	ClientID             string `mapstructure:"clientId"`
	ClientSecret         string `mapstructure:"clientSecret"`
	WebhookVerifierToken string `mapstructure:"webhookVerifierToken"`
	Environment          string `mapstructure:"environment"          validate:"omitempty,oneof=sandbox production"`
	RedirectURL          string `mapstructure:"redirectUrl"          validate:"omitempty,url"`
}

const quickBooksCallbackPath = "/admin/integrations/quickbooks/callback"

func (c *QuickBooksConfig) GetEnvironment() string {
	if c.Environment == "" {
		return "production"
	}
	return c.Environment
}

func (c *QuickBooksConfig) GetRedirectURL(app *appconfig.Config) string {
	if c.RedirectURL != "" {
		return c.RedirectURL
	}
	if base := app.GetWebBaseURL(); base != "" {
		return base + quickBooksCallbackPath
	}
	return ""
}

func (c *QuickBooksConfig) IsConfigured(app *appconfig.Config) bool {
	return strings.TrimSpace(c.ClientID) != "" &&
		strings.TrimSpace(c.ClientSecret) != "" &&
		c.GetRedirectURL(app) != ""
}
