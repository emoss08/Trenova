package useragentutils_test

import (
	"testing"

	"github.com/emoss08/trenova/shared/useragentutils"
	"github.com/stretchr/testify/assert"
)

func TestDescribe(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		ua    string
		label string
		kind  useragentutils.DeviceKind
	}{
		{
			"chrome on macos",
			"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36",
			"Chrome on macOS",
			useragentutils.DeviceDesktop,
		},
		{
			"edge on windows is not read as chrome",
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36 Edg/129.0.0.0",
			"Edge on Windows",
			useragentutils.DeviceDesktop,
		},
		{
			"safari on iphone",
			"Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Mobile/15E148 Safari/604.1",
			"Safari on iOS",
			useragentutils.DeviceMobile,
		},
		{
			"firefox on ubuntu",
			"Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:131.0) Gecko/20100101 Firefox/131.0",
			"Firefox on Ubuntu",
			useragentutils.DeviceDesktop,
		},
		{
			"chrome on an android tablet",
			"Mozilla/5.0 (Linux; Android 14; SM-X710) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36",
			"Chrome on Android",
			useragentutils.DeviceTablet,
		},
		{"empty", "", "", useragentutils.DeviceUnknown},
		{"unrecognised", "curl/8.4.0", "", useragentutils.DeviceUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := useragentutils.Describe(tt.ua)
			assert.Equal(t, tt.label, got.Label())
			assert.Equal(t, tt.kind, got.Device)
		})
	}
}
