package agentdefinition

import (
	"errors"
	"hash/fnv"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentmark"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttemplate"
)

var templateIcons = map[agenttemplate.Template]string{
	agenttemplate.TemplateDispatchAssistant:    agentmark.IconTruck,
	agenttemplate.TemplateBillingAssistant:     agentmark.IconReceipt,
	agenttemplate.TemplateComplianceAssistant:  agentmark.IconShield,
	agenttemplate.TemplateCustomerAssistant:    agentmark.IconHeadset,
	agenttemplate.TemplateGeneralAssistant:     agentmark.IconBot,
	agenttemplate.TemplateBillingException:     agentmark.IconReceipt,
	agenttemplate.TemplateDispatchAssignment:   agentmark.IconRoute,
	agenttemplate.TemplateImportAssistant:      agentmark.IconFile,
	agenttemplate.TemplateLoadMonitor:          agentmark.IconRadar,
	agenttemplate.TemplateShipmentIntake:       agentmark.IconPackage,
	agenttemplate.TemplateCashApplication:      agentmark.IconWallet,
	agenttemplate.TemplateDetentionDesk:        agentmark.IconGauge,
	agenttemplate.TemplateCredentialDesk:       agentmark.IconClipboard,
	agenttemplate.TemplateCustomerUpdateDesk:   agentmark.IconBell,
	agenttemplate.TemplateCarrierRiskDesk:      agentmark.IconSearch,
	agenttemplate.TemplateIntakeDesk:           agentmark.IconInbox,
	agenttemplate.TemplateLoadEntryCheck:       agentmark.IconClipboard,
	agenttemplate.TemplateServiceFailureDesk:   agentmark.IconHeadset,
	agenttemplate.TemplateInsightAnalyst:       agentmark.IconCompass,
	agenttemplate.TemplateEDIDesk:              agentmark.IconFile,
	agenttemplate.TemplateFormulaAssistant:     agentmark.IconSparkle,
	agenttemplate.TemplateBooksKeeper:          agentmark.IconReceipt,
	agenttemplate.TemplateFactoringClerk:       agentmark.IconBanknote,
	agenttemplate.TemplateSettlementsClerk:     agentmark.IconBanknote,
	agenttemplate.TemplateReceivables:          agentmark.IconCoins,
	agenttemplate.TemplateMasterDataSteward:    agentmark.IconFile,
	agenttemplate.TemplateWorkforceCoordinator: agentmark.IconClipboard,
	agenttemplate.TemplateFuelTaxClerk:         agentmark.IconGauge,
	agenttemplate.TemplateReportAnalyst:        agentmark.IconCompass,
}

func TemplateIcon(template agenttemplate.Template) string {
	return templateIcons[template]
}

// ResolvedIcon is what an agent should be drawn with: its own choice, else the
// icon its starter implies, else the generic one.
func (d *Definition) ResolvedIcon() string {
	if icon := d.ChosenIcon(); icon != "" {
		return icon
	}

	return agentmark.IconBot
}

// ChosenIcon is the agent's own icon, else the one its starter implies, and
// empty when it has neither. A reader that is handed only an agent's id and
// name draws an agent with no icon of its own by its initials, which the
// generic icon would hide, so this is what is served beside the name.
func (d *Definition) ChosenIcon() string {
	if icon := strings.TrimSpace(d.Icon); agentmark.IsKnownIcon(icon) {
		return icon
	}
	if icon, ok := templateIcons[d.Template]; ok {
		return icon
	}

	return ""
}

// ResolvedAccent is the agent's own accent, else one derived from its identity.
//
// The derivation hashes the id so an agent keeps the same colour for life and two
// agents in a list rarely collide. A definition that has not been saved yet falls
// back to its name so the builder can preview a mark before the first write.
func (d *Definition) ResolvedAccent() string {
	if accent := strings.TrimSpace(d.Accent); agentmark.IsKnownAccent(accent) {
		return accent
	}

	seed := d.ID.String()
	if strings.TrimSpace(seed) == "" {
		seed = strings.TrimSpace(d.Name)
	}

	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(seed))

	accents := agentmark.KnownAccents()

	return accents[int(hasher.Sum32()%uint32(len(accents)))]
}

var (
	errIconUnknown   = errors.New("Icon is not one this system offers")
	errAccentUnknown = errors.New("Accent is not one this system offers")
)
