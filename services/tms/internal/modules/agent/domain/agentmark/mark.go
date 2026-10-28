package agentmark

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
	IconBanknote  = "banknote"
	IconCoins     = "coins"
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
	IconBanknote,
	IconCoins,
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

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}

	return false
}
