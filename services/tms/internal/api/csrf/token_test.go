package csrf_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"testing"

	"github.com/emoss08/trenova/internal/api/csrf"
	"github.com/stretchr/testify/assert"
)

const secret = "unit-test-session-secret-with-32-bytes"

func TestTokenVerifiesForItsSessionOnly(t *testing.T) {
	t.Parallel()

	token := csrf.Token("ses_one", secret)

	assert.True(t, csrf.Verify(token, "ses_one", secret))
	assert.False(t, csrf.Verify(token, "ses_two", secret))
	assert.False(t, csrf.Verify(token, "ses_one", "another-session-secret-of-32-bytes!!"))
}

func TestTokenIsNotKeyedByTheRawSessionSecret(t *testing.T) {
	t.Parallel()

	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte("ses_one"))
	raw := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	assert.NotEqual(t, raw, csrf.Token("ses_one", secret))
}
