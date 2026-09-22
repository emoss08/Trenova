package agentdefinition

import (
	"errors"
	"hash/fnv"
	"strings"
)

const (
	IconBot       = "bot"
	IconTruck     = "truck"
	IconRoute     = "route"
	IconReceipt   = "receipt"
	IconWallet    = "wallet"
	IconShield    = "shield"
	IconHeadset   = "headset"
	IconClipboard = "clipboard"
	IconCompass   = "compass"
	IconRadar     = "radar"
	IconGauge     = "gauge"
	IconPackage   = "package"
	IconFile      = "file"
	IconSearch    = "search"
	IconBell      = "bell"
	IconSparkle   = "sparkle"
	IconInbox     = "inbox"
)

const (
	AccentIndigo = "indigo"
	AccentTeal   = "teal"
	AccentAmber  = "amber"
	AccentRose   = "rose"
	AccentEmeral = "emerald"
	AccentSky    = "sky"
	AccentViolet = "violet"
	AccentSlate  = "slate"
)

var knownIcons = []string{
	IconBot,
	IconTruck,
	IconRoute,
	IconReceipt,
	IconWallet,
	IconShield,
	IconHeadset,
	IconClipboard,
	IconCompass,
	IconRadar,
	IconGauge,
	IconPackage,
	IconFile,
	IconSearch,
	IconBell,
	IconSparkle,
	IconInbox,
}

var knownAccents = []string{
	AccentIndigo,
	AccentTeal,
	AccentAmber,
	AccentRose,
	AccentEmeral,
	AccentSky,
	AccentViolet,
	AccentSlate,
}

var templateIcons = map[Template]string{
	TemplateDispatchAssistant:   IconTruck,
	TemplateBillingAssistant:    IconReceipt,
	TemplateComplianceAssistant: IconShield,
	TemplateCustomerAssistant:   IconHeadset,
	TemplateGeneralAssistant:    IconBot,
	TemplateBillingException:    IconReceipt,
	TemplateDispatchAssignment:  IconRoute,
	TemplateImportAssistant:     IconFile,
	TemplateLoadMonitor:         IconRadar,
	TemplateShipmentIntake:      IconPackage,
	TemplateCashApplication:     IconWallet,
	TemplateDetentionDesk:       IconGauge,
	TemplateCredentialDesk:      IconClipboard,
	TemplateCustomerUpdateDesk:  IconBell,
	TemplateCarrierRiskDesk:     IconSearch,
	TemplateIntakeDesk:          IconInbox,
}

func KnownIcons() []string {
	out := make([]string, len(knownIcons))
	copy(out, knownIcons)

	return out
}

func KnownAccents() []string {
	out := make([]string, len(knownAccents))
	copy(out, knownAccents)

	return out
}

func IsKnownIcon(name string) bool {
	return contains(knownIcons, name)
}

func IsKnownAccent(name string) bool {
	return contains(knownAccents, name)
}

// ResolvedIcon is what an agent should be drawn with: its own choice, else the
// icon its starter implies, else the generic one.
func (d *Definition) ResolvedIcon() string {
	if icon := strings.TrimSpace(d.Icon); IsKnownIcon(icon) {
		return icon
	}
	if icon, ok := templateIcons[d.Template]; ok {
		return icon
	}

	return IconBot
}

// ResolvedAccent is the agent's own accent, else one derived from its identity.
//
// The derivation hashes the id so an agent keeps the same colour for life and two
// agents in a list rarely collide. A definition that has not been saved yet falls
// back to its name so the builder can preview a mark before the first write.
func (d *Definition) ResolvedAccent() string {
	if accent := strings.TrimSpace(d.Accent); IsKnownAccent(accent) {
		return accent
	}

	seed := d.ID.String()
	if strings.TrimSpace(seed) == "" {
		seed = strings.TrimSpace(d.Name)
	}

	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(seed))

	return knownAccents[int(hasher.Sum32()%uint32(len(knownAccents)))]
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}

	return false
}

var (
	errIconUnknown   = errors.New("Icon is not one this system offers")
	errAccentUnknown = errors.New("Accent is not one this system offers")
)
