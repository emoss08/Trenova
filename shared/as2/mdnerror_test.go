package as2

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMDNErrorForNeverRepeatsTheCause(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "a padding failure",
			err:  fmt.Errorf("%w: crypto/rsa: decryption error", ErrDecryptionFailed),
			want: MDNErrorDecryptionFailed,
		},
		{
			name: "a key of the wrong size",
			err:  fmt.Errorf("%w: crypto/aes: invalid key size 5", ErrDecryptionFailed),
			want: MDNErrorDecryptionFailed,
		},
		{
			name: "a signature that did not verify",
			err:  fmt.Errorf("%w: bad signer", ErrAuthenticationFailed),
			want: MDNErrorAuthenticationFailed,
		},
		{name: "no signature", err: ErrSignatureRequired, want: MDNErrorAuthenticationFailed},
		{
			name: "no certificate to verify with",
			err:  ErrNoPartnerCertificate,
			want: MDNErrorAuthenticationFailed,
		},
		{
			name: "a compression bomb",
			err:  fmt.Errorf("%w: too large", ErrDecompressionFailed),
			want: MDNErrorDecompressionFailed,
		},
		{name: "no encryption", err: ErrEncryptionRequired, want: MDNErrorInsufficientSecurity},
		{
			name: "a storage failure",
			err:  errors.New("pq: connection to 10.0.0.5 refused"),
			want: MDNErrorUnexpected,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tt.want, MDNErrorFor(tt.err))
		})
	}
}
