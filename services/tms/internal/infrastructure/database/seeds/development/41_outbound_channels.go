package development

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

const (
	SeedEmailProfileName = "Trenova Local"
	SeedPortalSiteName   = "Trenova App"
)

type OutboundChannelsSeed struct {
	seedhelpers.BaseSeed
}

func NewOutboundChannelsSeed() *OutboundChannelsSeed {
	seed := &OutboundChannelsSeed{}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"OutboundChannels",
		"1.0.0",
		"Adds an email profile with no key and a portal site with no sign-in, so agents can propose sending and portal work",
		[]common.Environment{common.EnvDevelopment},
	)
	seed.SetDependencies(seedhelpers.SeedAdminAccount)
	return seed
}

func (s *OutboundChannelsSeed) Run(ctx context.Context, tx bun.Tx) error {
	return seedhelpers.RunInTransaction(
		ctx,
		tx,
		s.Name(),
		nil,
		func(ctx context.Context, tx bun.Tx, sc *seedhelpers.SeedContext) error {
			org, err := sc.GetDefaultOrganization(ctx)
			if err != nil {
				return err
			}
			admin, err := sc.GetUserByUsername(ctx, "admin")
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
			inserted, err := insertSeedRowOnce(ctx, tx, sc, &seedRowOnce[email.Profile]{
				seedName: s.Name(),
				table:    "email_profiles",
				row:      profile,
				id:       func() pulid.ID { return profile.ID },
				where: func(q *bun.SelectQuery) *bun.SelectQuery {
					return q.Where("organization_id = ?", org.ID).
						Where("business_unit_id = ?", org.BusinessUnitID).
						Where("name = ?", SeedEmailProfileName)
				},
			})
			if err != nil {
				return err
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
			added, err := insertSeedRowOnce(ctx, tx, sc, &seedRowOnce[agentextension.PortalSite]{
				seedName: s.Name(),
				table:    "agent_portal_sites",
				row:      site,
				id:       func() pulid.ID { return site.ID },
				where: func(q *bun.SelectQuery) *bun.SelectQuery {
					return q.Where("organization_id = ?", org.ID).
						Where("business_unit_id = ?", org.BusinessUnitID).
						Where("name = ?", SeedPortalSiteName)
				},
			})
			if err != nil {
				return err
			}

			if inserted || added {
				seedhelpers.LogSuccess(
					"Added outbound channels",
					"- Email profile "+SeedEmailProfileName+" (no key, nothing is delivered)",
					"- Portal site "+SeedPortalSiteName+" (cloud.trenova.app, actions allowed)",
				)
			}

			return nil
		},
	)
}

func (s *OutboundChannelsSeed) Down(ctx context.Context, tx bun.Tx) error {
	return seedhelpers.RunInTransaction(
		ctx,
		tx,
		s.Name(),
		nil,
		func(ctx context.Context, tx bun.Tx, sc *seedhelpers.SeedContext) error {
			return seedhelpers.DeleteTrackedEntities(ctx, tx, s.Name(), sc)
		},
	)
}

func (s *OutboundChannelsSeed) CanRollback() bool {
	return true
}
