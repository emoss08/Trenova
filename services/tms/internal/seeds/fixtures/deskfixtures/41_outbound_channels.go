package deskfixtures

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/keel/seed"
	"github.com/emoss08/trenova/internal/modules/agent/domain/extensionportal"
	"github.com/emoss08/trenova/internal/modules/agent/domain/extensionportal/extensionportaldb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email/emaildb"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration/integrationdb"
	"github.com/emoss08/trenova/internal/seeds/fixtures/tenantfixtures"
	"github.com/emoss08/trenova/internal/seeds/seedenv"
)

const (
	SeedEmailProfileName = "Trenova Local"
	SeedPortalSiteName   = "Trenova App"
	// SeedMailSimulatorURL is where dev/mail-sim listens. The seeded Resend
	// integration sends there, so an email written in development is kept and
	// shown at that address and delivered to nobody.
	SeedMailSimulatorURL = "http://localhost:8092"
	seedMailSimulatorKey = "mail-sim"
)

// OutboundChannelsSeed gives a development organization a way to send: an
// email profile assigned to every purpose and a Resend integration pointed at
// the mail simulator, so a plan that ends in an email can run to its end. The
// simulator needs integrations.allowPrivateNetworks in the service's config.
// Each part is added only when it is missing, so the seed can be applied again.
//
// Depends on:
//   - AdminAccount: the default organization and its administrator
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

	profile, err := s.profile(ctx, db, org)
	if err != nil {
		return err
	}
	if err = s.assign(ctx, db, org, profile); err != nil {
		return err
	}
	if err = s.simulator(ctx, db, org, admin); err != nil {
		return err
	}

	return s.portalSite(ctx, db, org, admin)
}

func (s *OutboundChannelsSeed) profile(
	ctx context.Context,
	db *keel.DB,
	org *tenant.Organization,
) (*email.Profile, error) {
	p := emaildb.Profiles
	existing, err := p.AllTenants().Where(
		p.OrganizationID.Eq(org.ID),
		p.BusinessUnitID.Eq(org.BusinessUnitID),
		p.Name.Eq(SeedEmailProfileName),
	).First(ctx, db)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, keel.ErrNotFound) {
		return nil, fmt.Errorf("look for the email profile: %w", err)
	}

	profile := &email.Profile{
		OrganizationID: org.ID,
		BusinessUnitID: org.BusinessUnitID,
		Name:           SeedEmailProfileName,
		Description:    "Development sender: what it sends is kept by the mail simulator and delivered to nobody",
		SenderName:     "Trenova",
		SenderEmail:    "help@trenova.app",
		Provider:       email.ProviderResend,
		AuthType:       email.AuthTypeAPIKey,
		EncryptionType: email.EncryptionNone,
		Status:         email.ProfileStatusActive,
	}
	if err = p.Create(ctx, db, profile); err != nil {
		return nil, fmt.Errorf("insert email profile: %w", err)
	}

	return profile, nil
}

func (s *OutboundChannelsSeed) assign(
	ctx context.Context,
	db *keel.DB,
	org *tenant.Organization,
	profile *email.Profile,
) error {
	a := emaildb.ProfileAssignments
	assigned, err := a.AllTenants().Where(
		a.OrganizationID.Eq(org.ID),
		a.BusinessUnitID.Eq(org.BusinessUnitID),
		a.Purpose.Eq(email.PurposeGeneral),
	).Exists(ctx, db)
	if err != nil {
		return fmt.Errorf("look for the email profile's assignment: %w", err)
	}
	if assigned {
		return nil
	}
	if err = a.Create(ctx, db, &email.ProfileAssignment{
		OrganizationID: org.ID,
		BusinessUnitID: org.BusinessUnitID,
		Purpose:        email.PurposeGeneral,
		ProfileID:      profile.ID,
	}); err != nil {
		return fmt.Errorf("assign the email profile: %w", err)
	}

	return nil
}

// simulator points the organization's Resend integration at the mail
// simulator. The key it stores opens nothing: the simulator takes any. With
// encryption off the integration is left out, and email stays refused.
func (s *OutboundChannelsSeed) simulator(
	ctx context.Context,
	db *keel.DB,
	org *tenant.Organization,
	admin *tenant.User,
) error {
	i := integrationdb.Integrations
	configured, err := i.AllTenants().Where(
		i.OrganizationID.Eq(org.ID),
		i.BusinessUnitID.Eq(org.BusinessUnitID),
		i.Type.Eq(integration.TypeResend),
	).Exists(ctx, db)
	if err != nil {
		return fmt.Errorf("look for the Resend integration: %w", err)
	}
	if configured {
		return nil
	}

	sealer, err := encryptionservice.New(
		encryptionservice.Params{SecurityConfig: &seedenv.Config(ctx).Security},
	)
	if errors.Is(err, encryptionservice.ErrLocalKeyRequired) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("open the encryption service: %w", err)
	}
	key, err := sealer.EncryptStringWithAAD(seedMailSimulatorKey, encryptionservice.AAD{
		Purpose:        encryptionservice.PurposeIntegrationSecret,
		OrganizationID: org.ID,
		BusinessUnitID: org.BusinessUnitID,
		ResourceID:     "integration:" + integration.TypeResend.String() + ":apiKey",
	})
	if errors.Is(err, encryptionservice.ErrKeyManagerDisabled) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("seal the mail simulator's key: %w", err)
	}

	if err = i.Create(ctx, db, &integration.Integration{
		OrganizationID: org.ID,
		BusinessUnitID: org.BusinessUnitID,
		Type:           integration.TypeResend,
		Name:           "Resend",
		Description:    "Development: sends to the mail simulator at " + SeedMailSimulatorURL,
		Enabled:        true,
		Category:       integration.CategoryEmail,
		Configuration: map[string]any{
			"apiKey":  key,
			"baseUrl": SeedMailSimulatorURL,
		},
		EnabledByID: admin.ID,
	}); err != nil {
		return fmt.Errorf("insert the Resend integration: %w", err)
	}

	return nil
}

func (s *OutboundChannelsSeed) portalSite(
	ctx context.Context,
	db *keel.DB,
	org *tenant.Organization,
	admin *tenant.User,
) error {
	ps := extensionportaldb.PortalSites
	seeded, err := ps.AllTenants().Where(
		ps.OrganizationID.Eq(org.ID),
		ps.BusinessUnitID.Eq(org.BusinessUnitID),
		ps.Name.Eq(SeedPortalSiteName),
	).Exists(ctx, db)
	if err != nil {
		return fmt.Errorf("look for the portal site: %w", err)
	}
	if seeded {
		return nil
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
	if err = ps.Create(ctx, db, site); err != nil {
		return fmt.Errorf("insert portal site: %w", err)
	}

	return nil
}
