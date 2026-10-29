package deskfixtures

import (
	"context"
	"fmt"

	"github.com/emoss08/keel/seed"
	"github.com/emoss08/trenova/internal/modules/agent/domain/extensionportal"
	"github.com/emoss08/trenova/internal/modules/agent/domain/extensionportal/extensionportaldb"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email/emaildb"
	"github.com/emoss08/trenova/internal/seeds/fixtures/tenantfixtures"
)

const (
	SeedEmailProfileName = "Trenova Local"
	SeedPortalSiteName   = "Trenova App"
)

type OutboundChannelsSeed struct{}

func (s *OutboundChannelsSeed) Run(ctx context.Context, sr *seed.Run) error {
	db := sr.DB
	org, err := tenantfixtures.DefaultOrganization(ctx, sr)
	if err != nil {
		return err
	}
	admin, err := tenantfixtures.UserByUsername(ctx, db, "admin")
	if err != nil {
		return fmt.Errorf("get admin user: %w", err)
	}

	profile := &email.Profile{
		OrganizationID: org.ID,
		BusinessUnitID: org.BusinessUnitID,
		Name:           SeedEmailProfileName,
		Description:    "Development sender with no provider key: requests can be drafted and proposed, and none is delivered",
		SenderName:     "Trenova",
		SenderEmail:    "help@trenova.app",
		Provider:       email.ProviderResend,
		AuthType:       email.AuthTypeAPIKey,
		EncryptionType: email.EncryptionNone,
		Status:         email.ProfileStatusActive,
	}
	if err = emaildb.Profiles.Create(ctx, db, profile); err != nil {
		return fmt.Errorf("insert email profile: %w", err)
	}

	site := &extensionportal.PortalSite{
		OrganizationID: org.ID,
		BusinessUnitID: org.BusinessUnitID,
		Name:           SeedPortalSiteName,
		Purpose:        "Trenova's own web app: look up a load's status and update it for the team. Signs in with no saved credentials in development.",
		Domains:        []string{"cloud.trenova.app"},
		StartURL:       "https://cloud.trenova.app/",
		Credentials:    map[string]any{},
		AllowActions:   true,
		Enabled:        true,
		CreatedByID:    admin.ID,
	}
	if err = extensionportaldb.PortalSites.Create(ctx, db, site); err != nil {
		return fmt.Errorf("insert portal site: %w", err)
	}

	return nil
}
