package accountingsync

import (
	"slices"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
)

type ProviderProfile struct {
	Type                         integration.Type
	Name                         string
	WebhookSlug                  string
	CallbackPath                 string
	AppName                      string
	WebhookKeyLabel              string
	LineKind                     ReferenceKind
	ReferenceKinds               []ReferenceKind
	AccountRoles                 []string
	RequiredAccountRoles         []string
	Environments                 []AppEnvironment
	RefreshTokenLifetime         time.Duration
	RefreshTokenAbsoluteLifetime time.Duration
	Ledger                       bool
	LedgerUnavailableReason      string
	CallbackCarriesCompany       bool
	InboundPayments              bool
	InboundUnavailableReason     string
	WebhookSubscriptions         bool
	RevokesTokens                bool
}

var providerProfiles = []ProviderProfile{
	{
		Type:            integration.TypeQuickBooksOnline,
		Name:            "QuickBooks Online",
		WebhookSlug:     "quickbooks",
		CallbackPath:    "/admin/integrations/quickbooks/callback",
		AppName:         "Intuit app",
		WebhookKeyLabel: "Webhook verifier token",
		LineKind:        ReferenceKindItem,
		ReferenceKinds: []ReferenceKind{
			ReferenceKindAccount,
			ReferenceKindItem,
			ReferenceKindCustomer,
			ReferenceKindVendor,
			ReferenceKindTerm,
			ReferenceKindPaymentMethod,
		},
		AccountRoles: []string{
			AccountRoleAR,
			AccountRoleRevenue,
			AccountRoleDeposit,
			AccountRoleWriteOff,
			AccountRoleAP,
			AccountRolePurchasedTransportation,
		},
		RequiredAccountRoles: []string{
			AccountRoleAR,
			AccountRoleRevenue,
			AccountRoleDeposit,
		},
		Environments: []AppEnvironment{
			AppEnvironmentSandbox,
			AppEnvironmentProduction,
		},
		RefreshTokenLifetime:         100 * 24 * time.Hour,
		RefreshTokenAbsoluteLifetime: 5 * 365 * 24 * time.Hour,
		Ledger:                       true,
		CallbackCarriesCompany:       true,
		InboundPayments:              true,
		RevokesTokens:                true,
	},
	{
		Type:            integration.TypeXero,
		Name:            "Xero",
		WebhookSlug:     "xero",
		CallbackPath:    "/admin/integrations/xero/callback",
		AppName:         "Xero app",
		WebhookKeyLabel: "Webhook key",
		LineKind:        ReferenceKindAccount,
		ReferenceKinds: []ReferenceKind{
			ReferenceKindAccount,
			ReferenceKindItem,
			ReferenceKindCustomer,
			ReferenceKindVendor,
		},
		AccountRoles: []string{
			AccountRoleDeposit,
			AccountRolePurchasedTransportation,
		},
		RequiredAccountRoles: []string{
			AccountRoleDeposit,
		},
		Environments: []AppEnvironment{
			AppEnvironmentProduction,
		},
		RefreshTokenLifetime: 60 * 24 * time.Hour,
		Ledger:               false,
		LedgerUnavailableReason: "Xero manual journals cannot post to accounts receivable, " +
			"accounts payable or bank accounts, or name a customer or supplier, " +
			"so Trenova's journals cannot be sent to Xero as they are. Send documents instead.",
		CallbackCarriesCompany: false,
		InboundPayments:        true,
		RevokesTokens:          true,
	},
	{
		Type:            integration.TypeBusinessCentral,
		Name:            "Business Central",
		WebhookSlug:     "businesscentral",
		CallbackPath:    "/admin/integrations/business-central/callback",
		AppName:         "Microsoft Entra app",
		WebhookKeyLabel: "",
		LineKind:        ReferenceKindItem,
		ReferenceKinds: []ReferenceKind{
			ReferenceKindAccount,
			ReferenceKindItem,
			ReferenceKindCustomer,
			ReferenceKindVendor,
			ReferenceKindTerm,
		},
		AccountRoles: []string{
			AccountRoleDeposit,
			AccountRolePurchasedTransportation,
		},
		RequiredAccountRoles: []string{
			AccountRoleDeposit,
		},
		Environments: []AppEnvironment{
			AppEnvironmentProduction,
		},
		RefreshTokenLifetime: 90 * 24 * time.Hour,
		Ledger:               false,
		LedgerUnavailableReason: "Business Central journal lines cannot post to a customer or " +
			"vendor or apply to an invoice, so Trenova's journals cannot be sent to " +
			"Business Central as they are. Send documents instead.",
		CallbackCarriesCompany: false,
		InboundPayments:        false,
		InboundUnavailableReason: "Business Central's API does not let Trenova read posted " +
			"payments or which invoices they paid, so payments recorded in Business Central " +
			"are not brought in. An invoice paid there shows as a balance difference on the " +
			"drift page.",
		WebhookSubscriptions: true,
		RevokesTokens:        false,
	},
}

func Profile(typ integration.Type) (ProviderProfile, bool) {
	for idx := range providerProfiles {
		if providerProfiles[idx].Type == typ {
			return providerProfiles[idx], true
		}
	}
	return ProviderProfile{}, false
}

func MustProfile(typ integration.Type) ProviderProfile {
	profile, ok := Profile(typ)
	if !ok {
		return ProviderProfile{Type: typ, Name: string(typ)}
	}
	return profile
}

func AccountingSystems() []integration.Type {
	systems := make([]integration.Type, 0, len(providerProfiles))
	for idx := range providerProfiles {
		systems = append(systems, providerProfiles[idx].Type)
	}
	return systems
}

func ProfileByWebhookSlug(slug string) (ProviderProfile, bool) {
	for idx := range providerProfiles {
		if providerProfiles[idx].WebhookSlug == slug {
			return providerProfiles[idx], true
		}
	}
	return ProviderProfile{}, false
}

func (p *ProviderProfile) Has(kind ReferenceKind) bool {
	return slices.Contains(p.ReferenceKinds, kind)
}

func (p *ProviderProfile) KindFor(target MappingTargetType) ReferenceKind {
	var kind ReferenceKind
	switch target {
	case TargetAccountRole, TargetGLAccount:
		kind = ReferenceKindAccount
	case TargetLineType, TargetAccessorialCharge, TargetItemRole:
		kind = p.LineKind
	case TargetCustomer:
		kind = ReferenceKindCustomer
	case TargetCarrier, TargetDriver:
		kind = ReferenceKindVendor
	case TargetPaymentTerm:
		kind = ReferenceKindTerm
	case TargetPaymentMethod:
		kind = ReferenceKindPaymentMethod
	}
	if kind == "" || !p.Has(kind) {
		return ""
	}
	return kind
}

func (p *ProviderProfile) Offers(target MappingTargetType) bool {
	return p.KindFor(target) != ""
}

func (p *ProviderProfile) OffersRole(role string) bool {
	return slices.Contains(p.AccountRoles, role)
}

func (p *ProviderProfile) OffersKey(target MappingTargetType, key string) bool {
	if !p.Offers(target) {
		return false
	}
	if target == TargetAccountRole {
		return p.OffersRole(key)
	}
	return true
}

func (p *ProviderProfile) IsRequiredTarget(target MappingTargetType, key string) bool {
	switch target { //nolint:exhaustive // only roles and the freight line are ever required
	case TargetAccountRole:
		return slices.Contains(p.RequiredAccountRoles, key)
	case TargetLineType:
		return p.Offers(target) && key == string(invoice.InvoiceLineTypeFreight)
	default:
		return false
	}
}

func (p *ProviderProfile) HasEnvironment(env AppEnvironment) bool {
	return slices.Contains(p.Environments, env)
}

func (p *ProviderProfile) DefaultEnvironment() AppEnvironment {
	if p.HasEnvironment(AppEnvironmentProduction) || len(p.Environments) == 0 {
		return AppEnvironmentProduction
	}
	return p.Environments[0]
}

func (p *ProviderProfile) WebhookPath() string {
	if p.WebhookSlug == "" {
		return ""
	}
	return "/webhooks/accounting/" + p.WebhookSlug + "/"
}

func (p *ProviderProfile) AppWebhookPath(appID string) string {
	base := p.WebhookPath()
	if base == "" || appID == "" {
		return base
	}
	return base + appID + "/"
}

func (p *ProviderProfile) SupportsMode(mode SyncMode) bool {
	if mode == SyncModeLedger {
		return p.Ledger
	}
	return mode.IsValid()
}

type TargetKey struct {
	Type MappingTargetType
	Key  string
}

func (p *ProviderProfile) RequiredTargets() []TargetKey {
	keys := make([]TargetKey, 0, len(p.RequiredAccountRoles)+1)
	for _, target := range AllMappingTargetTypes() {
		for _, key := range target.Keys() {
			if p.IsRequiredTarget(target, key) {
				keys = append(keys, TargetKey{Type: target, Key: key})
			}
		}
	}
	return keys
}
