package fixtures

import (
	"context"
	"fmt"

	"github.com/emoss08/keel/seed"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension/agentextensiondb"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email/emaildb"
)

const (
	SeedEmailProfileName = "Trenova Local"
	SeedPortalSiteName   = "Trenova App"
)

type OutboundChannelsSeed struct{}

var OutboundChannels = Seeds.Add(seed.Seed{
	Name:        "outbound_channels",
	Kind:        seed.Fixture,
	Description: "Adds an email profile with no key and a portal site with no sign-in, so agents can propose sending and portal work",
	After:       []*seed.Seed{AdminAccount},
	Run:         (&OutboundChannelsSeed{}).Run,
})

func (s *OutboundChannelsSeed) Run(ctx context.Context, sr *seed.Run) error {
	db := sr.DB
	org, err := defaultOrganization(ctx, sr)
	if err != nil {
		return err
	}
	admin, err := userByUsername(ctx, db, "admin")
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

	site := &agentextension.PortalSite{
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
	if err = agentextensiondb.PortalSites.Create(ctx, db, site); err != nil {
		return fmt.Errorf("insert portal site: %w", err)
	}

	return nil
}
