package accountingconnectionflows

import (
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingconnectionapps"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingconnectionauth"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingconnectioncert"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingconnectionhealth"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingconnectionkit"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingconnectionstatus"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingconnectionsubscriptions"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingconnectionwebhook"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectionports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountinginboundports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenteventports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/integrationports"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"go.uber.org/zap"
)

type wiring struct {
	Connections   accountingconnectionports.AccountingConnectionRepository
	Apps          accountingconnectionports.AccountingAppCredentialRepository
	States        accountingconnectionports.AccountingOAuthStateRepository
	Integrations  integrationports.IntegrationRepository
	Connectors    accountingconnectorports.AccountingConnectorRegistry
	Encryption    *encryptionservice.Service
	Audit         auditports.AuditService
	Watchtower    watchtowerports.WatchtowerProjector
	Publisher     agenteventports.AgentEventPublisher
	Refresher     accountingconnectionports.AccountingReferenceRefresher
	Poller        accountinginboundports.AccountingChangePoller
	Subscriptions accountingconnectionports.AccountingWebhookSubscriptionRepository
	Certificates  accountingconnectionports.AccountingPendingCertificateRepository
}

type services struct {
	health   *accountingconnectionhealth.Service
	subs     *accountingconnectionsubscriptions.Service
	status   *accountingconnectionstatus.Service
	apps     *accountingconnectionapps.Service
	auth     *accountingconnectionauth.Service
	certs    *accountingconnectioncert.Service
	webhooks *accountingconnectionwebhook.Service
}

func wire(w *wiring) services {
	l := zap.NewNop()
	db := dbtest.NopConnection{}
	kit := accountingconnectionkit.New(accountingconnectionkit.Params{
		Logger:       l,
		DB:           db,
		Connections:  w.Connections,
		Apps:         w.Apps,
		Integrations: w.Integrations,
		Connectors:   w.Connectors,
		Encryption:   w.Encryption,
		Audit:        w.Audit,
		Watchtower:   w.Watchtower,
		Publisher:    w.Publisher,
		Refresher:    w.Refresher,
		Certificates: w.Certificates,
	})
	health := accountingconnectionhealth.New(accountingconnectionhealth.Params{
		Logger:      l,
		DB:          db,
		Connections: w.Connections,
		Kit:         kit,
	})
	subs := accountingconnectionsubscriptions.New(accountingconnectionsubscriptions.Params{
		Logger:        l,
		Subscriptions: w.Subscriptions,
		Encryption:    w.Encryption,
		Kit:           kit,
		Health:        health,
	})
	status := accountingconnectionstatus.New(accountingconnectionstatus.Params{
		Connections: w.Connections,
		Kit:         kit,
		Subs:        subs,
	})
	auth := accountingconnectionauth.New(accountingconnectionauth.Params{
		Logger:       l,
		DB:           db,
		Connections:  w.Connections,
		States:       w.States,
		Certificates: w.Certificates,
		Encryption:   w.Encryption,
		Kit:          kit,
		Subs:         subs,
	})
	return services{
		health: health,
		subs:   subs,
		status: status,
		apps: accountingconnectionapps.New(accountingconnectionapps.Params{
			Logger:      l,
			DB:          db,
			Apps:        w.Apps,
			Connections: w.Connections,
			Encryption:  w.Encryption,
			Audit:       w.Audit,
			Kit:         kit,
			Status:      status,
		}),
		auth: auth,
		certs: accountingconnectioncert.New(accountingconnectioncert.Params{
			Logger:       l,
			DB:           db,
			Connections:  w.Connections,
			Certificates: w.Certificates,
			Encryption:   w.Encryption,
			Kit:          kit,
			Auth:         auth,
		}),
		webhooks: accountingconnectionwebhook.New(accountingconnectionwebhook.Params{
			Logger:        l,
			Apps:          w.Apps,
			Connections:   w.Connections,
			Subscriptions: w.Subscriptions,
			Encryption:    w.Encryption,
			Poller:        w.Poller,
			Kit:           kit,
		}),
	}
}
