package useragentutils

import "strings"

type DeviceKind string

const (
	DeviceDesktop DeviceKind = "desktop"
	DeviceMobile  DeviceKind = "mobile"
	DeviceTablet  DeviceKind = "tablet"
	DeviceUnknown DeviceKind = "unknown"
)

type Description struct {
	Browser string     `json:"browser"`
	OS      string     `json:"os"`
	Device  DeviceKind `json:"device"`
}

type marker struct {
	token string
	name  string
}

func browsers() []marker {
	return []marker{
		{"edg/", "Edge"},
		{"edga/", "Edge"},
		{"edgios/", "Edge"},
		{"opr/", "Opera"},
		{"opera", "Opera"},
		{"samsungbrowser/", "Samsung Internet"},
		{"vivaldi/", "Vivaldi"},
		{"brave/", "Brave"},
		{"firefox/", "Firefox"},
		{"fxios/", "Firefox"},
		{"crios/", "Chrome"},
		{"chrome/", "Chrome"},
		{"chromium/", "Chromium"},
		{"safari/", "Safari"},
	}
}

func systems() []marker {
	return []marker{
		{"iphone", "iOS"},
		{"ipad", "iPadOS"},
		{"ipod", "iOS"},
		{"android", "Android"},
		{"cros", "ChromeOS"},
		{"windows", "Windows"},
		{"mac os x", "macOS"},
		{"macintosh", "macOS"},
		{"ubuntu", "Ubuntu"},
		{"fedora", "Fedora"},
		{"linux", "Linux"},
	}
}

func Describe(userAgent string) Description {
	ua := strings.ToLower(strings.TrimSpace(userAgent))
	if ua == "" {
		return Description{Device: DeviceUnknown}
	}

	return Description{
		Browser: firstMatch(ua, browsers()),
		OS:      firstMatch(ua, systems()),
		Device:  deviceKind(ua),
	}
}

func (d Description) Label() string {
	switch {
	case d.Browser != "" && d.OS != "":
		return d.Browser + " on " + d.OS
	case d.Browser != "":
		return d.Browser
	case d.OS != "":
		return d.OS
	default:
		return ""
	}
}

func firstMatch(ua string, markers []marker) string {
	for _, m := range markers {
		if strings.Contains(ua, m.token) {
			return m.name
		}
	}
	return ""
}

func deviceKind(ua string) DeviceKind {
	switch {
	case strings.Contains(ua, "ipad") || strings.Contains(ua, "tablet") ||
		(strings.Contains(ua, "android") && !strings.Contains(ua, "mobile")):
		return DeviceTablet
	case strings.Contains(ua, "mobi") || strings.Contains(ua, "iphone") ||
		strings.Contains(ua, "ipod"):
		return DeviceMobile
	case strings.Contains(ua, "windows") || strings.Contains(ua, "macintosh") ||
		strings.Contains(ua, "linux") || strings.Contains(ua, "cros"):
		return DeviceDesktop
	default:
		return DeviceUnknown
	}
}
