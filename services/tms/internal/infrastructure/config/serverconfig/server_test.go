package serverconfig

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateTrustedProxies(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		proxies []string
		wantErr bool
	}{
		{name: "defaults are valid", proxies: DefaultTrustedProxies},
		{name: "empty trusts nothing", proxies: []string{}},
		{name: "bare addresses", proxies: []string{"192.0.2.10", "2001:db8::1"}},
		{name: "rejects a hostname", proxies: []string{"proxy.internal"}, wantErr: true},
		{name: "rejects a malformed cidr", proxies: []string{"10.0.0.0/64"}, wantErr: true},
		{name: "rejects a blank entry", proxies: []string{"  "}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := Config{TrustedProxies: tt.proxies}
			err := cfg.validateTrustedProxies()

			if tt.wantErr {
				require.ErrorIs(t, err, ErrInvalidTrustedProxy)
				return
			}
			require.NoError(t, err)
		})
	}
}
