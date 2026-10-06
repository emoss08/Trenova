package emailutils

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseNormalizes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		input      string
		lower      string
		normalized string
		domain     string
	}{
		{
			name:       "plain address is lower cased",
			input:      "  Dana.Whitfield@Example.COM ",
			lower:      "dana.whitfield@example.com",
			normalized: "dana.whitfield@example.com",
			domain:     "example.com",
		},
		{
			name:       "gmail dots and tag are stripped",
			input:      "Dana.Whit.Field+trenova@gmail.com",
			lower:      "dana.whit.field+trenova@gmail.com",
			normalized: "danawhitfield@gmail.com",
			domain:     "gmail.com",
		},
		{
			name:       "googlemail folds into gmail",
			input:      "d.w+x@googlemail.com",
			lower:      "d.w+x@googlemail.com",
			normalized: "dw@gmail.com",
			domain:     "googlemail.com",
		},
		{
			name:       "plus tags survive outside gmail",
			input:      "ops+billing@fleet.example.co",
			lower:      "ops+billing@fleet.example.co",
			normalized: "ops+billing@fleet.example.co",
			domain:     "fleet.example.co",
		},
		{
			name:       "punycode top level is accepted",
			input:      "a@example.xn--p1ai",
			lower:      "a@example.xn--p1ai",
			normalized: "a@example.xn--p1ai",
			domain:     "example.xn--p1ai",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			addr, err := Parse(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.lower, addr.Lower)
			assert.Equal(t, tt.normalized, addr.Normalized)
			assert.Equal(t, tt.domain, addr.Domain)
			assert.Equal(t, strings.TrimSpace(tt.input), addr.Original)

			normalized, err := Normalize(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.normalized, normalized)
		})
	}
}

func TestParseRejects(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  error
	}{
		{name: "empty", input: "   ", want: ErrEmpty},
		{name: "no at", input: "dana.example.com", want: ErrInvalidSyntax},
		{name: "display name", input: "Dana <dana@example.com>", want: ErrInvalidSyntax},
		{name: "space inside", input: "da na@example.com", want: ErrInvalidSyntax},
		{name: "leading dot", input: ".dana@example.com", want: ErrInvalidSyntax},
		{name: "double dot", input: "da..na@example.com", want: ErrInvalidSyntax},
		{name: "single label domain", input: "dana@localhost", want: ErrInvalidDomain},
		{name: "ip literal", input: "dana@[127.0.0.1]", want: ErrInvalidSyntax},
		{name: "numeric tld", input: "dana@example.123", want: ErrInvalidDomain},
		{name: "hyphen label", input: "dana@-example.com", want: ErrInvalidDomain},
		{name: "underscore domain", input: "dana@ex_ample.com", want: ErrInvalidDomain},
		{name: "empty gmail after tag", input: "+tag@gmail.com", want: ErrInvalidSyntax},
		{
			name:  "too long",
			input: strings.Repeat("a", 64) + "@" + strings.Repeat("b", 190) + ".com",
			want:  ErrTooLong,
		},
		{
			name:  "local part too long",
			input: strings.Repeat("a", 65) + "@example.com",
			want:  ErrInvalidSyntax,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := Parse(tt.input)
			require.Error(t, err)
			if errors.Is(tt.want, ErrInvalidSyntax) {
				assert.True(
					t,
					errors.Is(err, ErrInvalidSyntax) || errors.Is(err, ErrInvalidLocalPart),
					"got %v",
					err,
				)
				return
			}
			assert.ErrorIs(t, err, tt.want)
		})
	}
}

func TestDomainAndLocalPart(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "example.com", Domain(" Dana@Example.com "))
	assert.Equal(t, "dana", LocalPart(" Dana@Example.com "))
	assert.Empty(t, Domain("nobody"))
	assert.Empty(t, LocalPart("@example.com"))
}

func TestDomainAllowed(t *testing.T) {
	t.Parallel()

	assert.True(t, DomainAllowed("example.com", nil))
	assert.True(t, DomainAllowed("example.com", []string{"Example.com"}))
	assert.True(t, DomainAllowed("ops.example.com", []string{"example.com"}))
	assert.False(t, DomainAllowed("badexample.com", []string{"example.com"}))
	assert.False(t, DomainAllowed("example.org", []string{"example.com", " "}))
}

func TestIsDisposable(t *testing.T) {
	t.Parallel()

	assert.Greater(t, DisposableDomainCount(), 500)
	assert.True(t, IsDisposable("someone@mailinator.com"))
	assert.True(t, IsDisposable("someone@MAILINATOR.com"))
	assert.True(t, IsDisposableDomain("inbox.guerrillamail.com"))
	assert.True(t, IsDisposableDomain("yopmail.com."))
	assert.False(t, IsDisposable("someone@gmail.com"))
	assert.False(t, IsDisposable("someone@trenova.app"))
	assert.False(t, IsDisposableDomain(""))
	assert.False(t, IsDisposableDomain("com"))
}

func TestDisposableListIsClean(t *testing.T) {
	t.Parallel()

	for domain := range disposableDomains() {
		assert.NoError(t, ValidateDomain(domain), domain)
		assert.NotContains(t, []string{"gmail.com", "outlook.com", "yahoo.com"}, domain)
	}
}
