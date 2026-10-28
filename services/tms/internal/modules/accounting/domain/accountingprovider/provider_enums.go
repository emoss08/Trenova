package accountingprovider

import (
	"strings"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
)

func ParseAppEnvironment(value string) (AppEnvironment, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "sandbox":
		return AppEnvironmentSandbox, true
	case "production":
		return AppEnvironmentProduction, true
	default:
		return "", false
	}
}

func WebhookPath(typ integration.Type) string {
	profile := MustProfile(typ)
	return profile.WebhookPath()
}

func SupportsAccountingSync(typ integration.Type) bool {
	_, ok := Profile(typ)
	return ok
}

func ProviderName(typ integration.Type) string {
	return MustProfile(typ).Name
}

func SetupPath(typ integration.Type) string {
	return "/admin/integrations?type=" + string(typ)
}

type AppEnvironment string

const (
	AppEnvironmentSandbox    = AppEnvironment("Sandbox")
	AppEnvironmentProduction = AppEnvironment("Production")
)

func (e AppEnvironment) String() string { return string(e) }

func (e AppEnvironment) IsValid() bool {
	switch e {
	case AppEnvironmentSandbox, AppEnvironmentProduction:
		return true
	default:
		return false
	}
}

type SyncMode string

const (
	SyncModeDocument = SyncMode("Document")
	SyncModeLedger   = SyncMode("Ledger")
)

func (m SyncMode) String() string { return string(m) }

func (m SyncMode) IsValid() bool {
	switch m {
	case SyncModeDocument, SyncModeLedger:
		return true
	default:
		return false
	}
}

func AllSyncModes() []SyncMode {
	return []SyncMode{SyncModeDocument, SyncModeLedger}
}

func IsRequiredTarget(target accountingsync.MappingTargetType, key string) bool {
	for idx := range providerProfiles {
		if providerProfiles[idx].IsRequiredTarget(target, key) {
			return true
		}
	}
	return false
}

func WebhookSubscriptionSystems() []integration.Type {
	systems := make([]integration.Type, 0, len(providerProfiles))
	for idx := range providerProfiles {
		if providerProfiles[idx].WebhookSubscriptions {
			systems = append(systems, providerProfiles[idx].Type)
		}
	}
	return systems
}
