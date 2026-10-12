package businesscentral

import (
	"context"
	"net/http"
	"strings"
	"time"
)

const (
	ApplicationFamilyBusinessCentral = "BusinessCentral"
	EnvironmentTypeProduction        = "Production"
	EnvironmentTypeSandbox           = "Sandbox"
	companiesEntity                  = "companies"
)

type Environment struct {
	TenantID          string
	ApplicationFamily string
	Type              string
	Name              string
	CountryCode       string
	WebServiceURL     string
	WebClientLoginURL string
}

func (e *Environment) IsProduction() bool {
	return strings.EqualFold(e.Type, EnvironmentTypeProduction)
}

type Company struct {
	ID                string
	SystemVersion     string
	Timestamp         int64
	Name              string
	DisplayName       string
	BusinessProfileID string
	CreatedAt         time.Time
	ModifiedAt        time.Time
}

type DiscoveryClient struct {
	core *core
}

type wireEnvironment struct {
	AADTenantID       string `json:"aadTenantId"`
	ApplicationFamily string `json:"applicationFamily"`
	Type              string `json:"type"`
	Name              string `json:"name"`
	CountryCode       string `json:"countryCode"`
	WebServiceURL     string `json:"webServiceUrl"`
	WebClientLoginURL string `json:"webClientLoginUrl"`
}

type wireCompany struct {
	ID                string   `json:"id"`
	SystemVersion     string   `json:"systemVersion"`
	Timestamp         int64    `json:"timestamp"`
	Name              string   `json:"name"`
	DisplayName       string   `json:"displayName"`
	BusinessProfileID string   `json:"businessProfileId"`
	SystemCreatedAt   wireTime `json:"systemCreatedAt"`
	SystemModifiedAt  wireTime `json:"systemModifiedAt"`
}

func (w *wireCompany) company() Company {
	return Company{
		ID:                strings.ToLower(w.ID),
		SystemVersion:     w.SystemVersion,
		Timestamp:         w.Timestamp,
		Name:              w.Name,
		DisplayName:       w.DisplayName,
		BusinessProfileID: w.BusinessProfileID,
		CreatedAt:         w.SystemCreatedAt.time(),
		ModifiedAt:        w.SystemModifiedAt.time(),
	}
}

func NewDiscoveryClient(accessToken string, opts ...Option) (*DiscoveryClient, error) {
	token := strings.TrimSpace(accessToken)
	if token == "" {
		return nil, ErrAccessTokenRequired
	}
	settings := resolveOptions(opts)
	c, err := newCore(&settings, token, nil)
	if err != nil {
		return nil, err
	}
	return &DiscoveryClient{core: c}, nil
}

func (c *DiscoveryClient) Environments(ctx context.Context) ([]Environment, error) {
	var out collection[wireEnvironment]
	if _, err := c.core.do(ctx, &call{
		endpoint: "environments",
		method:   http.MethodGet,
		path:     environmentsPath,
		out:      &out,
	}); err != nil {
		return nil, err
	}

	environments := make([]Environment, 0, len(out.Value))
	for idx := range out.Value {
		wire := &out.Value[idx]
		family := strings.TrimSpace(wire.ApplicationFamily)
		if family != "" && !strings.EqualFold(family, ApplicationFamilyBusinessCentral) {
			continue
		}
		environments = append(environments, Environment{
			TenantID:          strings.ToLower(strings.TrimSpace(wire.AADTenantID)),
			ApplicationFamily: family,
			Type:              wire.Type,
			Name:              wire.Name,
			CountryCode:       wire.CountryCode,
			WebServiceURL:     wire.WebServiceURL,
			WebClientLoginURL: wire.WebClientLoginURL,
		})
	}
	return environments, nil
}

func (c *DiscoveryClient) Companies(
	ctx context.Context,
	tenantID, environment string,
) ([]Company, error) {
	tenant, err := guid(tenantID)
	if err != nil {
		return nil, err
	}
	env := strings.TrimSpace(environment)
	if !ValidEnvironmentName(env) {
		return nil, ErrInvalidEnvironment
	}
	return collect(ctx, c.core, &listCall{
		endpoint: "companies",
		path:     environmentPath(tenant, env) + "/" + companiesEntity,
	}, (*wireCompany).company)
}
