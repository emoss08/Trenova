package aiprovider_test

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/domain/editchange"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidate_DefaultsTheLimits(t *testing.T) {
	t.Parallel()

	provider := validProvider()
	require.Empty(t, fieldErrors(t, provider))

	assert.Equal(t, aiprovider.DefaultTimeoutSeconds, provider.TimeoutSeconds)
	assert.Equal(t, aiprovider.DefaultMaxConcurrent, provider.MaxConcurrent)
	assert.Equal(t, aiprovider.CapActionNext, provider.OnCap)
	assert.Nil(t, provider.MonthlyCapUSD)
}

func TestValidate_TimeoutMustBeBetweenFiveSecondsAndTenMinutes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		seconds int
		invalid bool
	}{
		{4, true},
		{5, false},
		{600, false},
		{601, true},
		{-1, true},
	}
	for _, tc := range cases {
		provider := validProvider()
		provider.TimeoutSeconds = tc.seconds
		assert.Equal(t, tc.invalid, fieldErrors(t, provider)["timeoutSeconds"], "timeout %d", tc.seconds)
	}
}

func TestValidate_ConcurrencyMustBeBetweenOneAndSixtyFour(t *testing.T) {
	t.Parallel()

	cases := []struct {
		calls   int
		invalid bool
	}{
		{1, false},
		{64, false},
		{65, true},
		{-3, true},
	}
	for _, tc := range cases {
		provider := validProvider()
		provider.MaxConcurrent = tc.calls
		assert.Equal(t, tc.invalid, fieldErrors(t, provider)["maxConcurrent"], "calls %d", tc.calls)
	}
}

func TestValidate_MonthlyCapMustBePositiveWithCents(t *testing.T) {
	t.Parallel()

	cases := []struct {
		cap     string
		invalid bool
	}{
		{"0", true},
		{"-5", true},
		{"0.01", false},
		{"250.50", false},
		{"10.005", true},
		{"10000000000", true},
	}
	for _, tc := range cases {
		provider := validProvider()
		limit := decimal.RequireFromString(tc.cap)
		provider.MonthlyCapUSD = &limit
		assert.Equal(t, tc.invalid, fieldErrors(t, provider)["monthlyCapUsd"], "cap %s", tc.cap)
	}
}

func TestValidate_OnCapMustBeNextOrStop(t *testing.T) {
	t.Parallel()

	provider := validProvider()
	provider.OnCap = aiprovider.CapAction("Pause")
	assert.True(t, fieldErrors(t, provider)["onCap"])

	provider.OnCap = aiprovider.CapActionStop
	assert.False(t, fieldErrors(t, provider)["onCap"])
}

func TestCapReached_ComparesSpendWithTheCap(t *testing.T) {
	t.Parallel()

	provider := validProvider()
	assert.False(t, provider.Capped())
	assert.False(t, provider.CapReached(decimal.RequireFromString("1000000")))

	limit := decimal.RequireFromString("50")
	provider.MonthlyCapUSD = &limit
	assert.True(t, provider.Capped())
	assert.False(t, provider.CapReached(decimal.RequireFromString("49.99")))
	assert.True(t, provider.CapReached(decimal.RequireFromString("50")))
	assert.True(t, provider.CapReached(decimal.RequireFromString("50.01")))
}

func TestResolvedLimits_FallBackToTheDefaults(t *testing.T) {
	t.Parallel()

	provider := &aiprovider.Provider{}
	assert.Equal(t, time.Duration(aiprovider.DefaultTimeoutSeconds)*time.Second, provider.ResolvedTimeout())
	assert.Equal(t, aiprovider.DefaultMaxConcurrent, provider.ResolvedMaxConcurrent())
	assert.Equal(t, aiprovider.CapActionNext, provider.ResolvedOnCap())

	provider.TimeoutSeconds = 30
	provider.MaxConcurrent = 2
	provider.OnCap = aiprovider.CapActionStop
	assert.Equal(t, 30*time.Second, provider.ResolvedTimeout())
	assert.Equal(t, 2, provider.ResolvedMaxConcurrent())
	assert.Equal(t, aiprovider.CapActionStop, provider.ResolvedOnCap())
}

func TestReplaceAPIKey_KeepsTheReplacedKeyForADayWhenAsked(t *testing.T) {
	t.Parallel()

	provider := validProvider()
	provider.APIKey = "enc-old"
	user := pulid.MustNew("usr_")
	now := int64(1_800_000_000)

	provider.ReplaceAPIKey(aiprovider.KeyReplacement{
		Encrypted:    "enc-new",
		Prefix:       "sk-ant-",
		LastFour:     "9876",
		AddedByID:    user,
		Now:          now,
		KeepPrevious: true,
	})

	assert.Equal(t, "enc-new", provider.APIKey)
	assert.Equal(t, "enc-old", provider.PreviousAPIKey)
	require.NotNil(t, provider.RotationExpiresAt)
	assert.Equal(t, now+int64(aiprovider.RotationGrace/time.Second), *provider.RotationExpiresAt)
	assert.Equal(t, "sk-ant-", provider.APIKeyPrefix)
	assert.Equal(t, "9876", provider.APIKeyLastFour)
	assert.Equal(t, user, provider.APIKeyAddedByID)
	require.NotNil(t, provider.APIKeyAddedAt)
	assert.Equal(t, now, *provider.APIKeyAddedAt)
	assert.True(t, provider.PreviousKeyUsable(now+1))
	assert.False(t, provider.PreviousKeyUsable(*provider.RotationExpiresAt))
}

func TestReplaceAPIKey_DropsThePreviousKeyUnlessAsked(t *testing.T) {
	t.Parallel()

	provider := validProvider()
	provider.APIKey = "enc-old"
	expires := int64(1_800_000_100)
	provider.PreviousAPIKey = "enc-older"
	provider.RotationExpiresAt = &expires
	used := int64(1_799_999_000)
	provider.APIKeyLastUsedAt = &used

	provider.ReplaceAPIKey(aiprovider.KeyReplacement{
		Encrypted: "enc-new",
		LastFour:  "1234",
		Now:       1_800_000_000,
	})

	assert.Empty(t, provider.PreviousAPIKey)
	assert.Nil(t, provider.RotationExpiresAt)
	assert.Nil(t, provider.APIKeyLastUsedAt, "a new key has not been used yet")
}

func TestReplaceAPIKey_ClearingTheKeyForgetsEverythingAboutIt(t *testing.T) {
	t.Parallel()

	provider := validProvider()
	provider.ReplaceAPIKey(aiprovider.KeyReplacement{
		Encrypted: "enc-old",
		Prefix:    "sk-",
		LastFour:  "1111",
		AddedByID: pulid.MustNew("usr_"),
		Now:       1_800_000_000,
	})
	provider.ReplaceAPIKey(aiprovider.KeyReplacement{Now: 1_800_000_500, KeepPrevious: true})

	assert.Empty(t, provider.APIKey)
	assert.Empty(t, provider.PreviousAPIKey, "clearing the key never keeps the old one")
	assert.Empty(t, provider.APIKeyPrefix)
	assert.Empty(t, provider.APIKeyLastFour)
	assert.Nil(t, provider.APIKeyAddedAt)
	assert.True(t, provider.APIKeyAddedByID.IsNil())
	assert.Nil(t, provider.KeyInfo(1_800_000_600))
}

func TestForgetExpiredPreviousKey_ClearsOnlyAnExpiredRotation(t *testing.T) {
	t.Parallel()

	provider := validProvider()
	expires := int64(1_000)
	provider.PreviousAPIKey = "enc-old"
	provider.RotationExpiresAt = &expires

	assert.False(t, provider.ForgetExpiredPreviousKey(999))
	assert.Equal(t, "enc-old", provider.PreviousAPIKey)

	assert.True(t, provider.ForgetExpiredPreviousKey(1_000))
	assert.Empty(t, provider.PreviousAPIKey)
	assert.Nil(t, provider.RotationExpiresAt)
}

func TestKeyInfo_DescribesTheKeyWithoutTheSecret(t *testing.T) {
	t.Parallel()

	provider := validProvider()
	provider.CreatedAt = 1_700_000_000
	provider.APIKey = "enc"
	provider.APIKeyLastFour = "4321"
	provider.APIKeyPrefix = "gsk_"
	expires := int64(1_800_000_100)
	provider.PreviousAPIKey = "enc-old"
	provider.RotationExpiresAt = &expires

	info := provider.KeyInfo(1_800_000_000)
	require.NotNil(t, info)
	assert.Equal(t, "gsk_", info.Prefix)
	assert.Equal(t, "4321", info.LastFour)
	assert.Equal(t, int64(1_700_000_000), info.AddedAt, "a key stored before its ends were recorded reads as added with the provider")
	require.NotNil(t, info.PreviousKeyExpiresAt)
	assert.Equal(t, expires, *info.PreviousKeyExpiresAt)

	assert.Nil(t, provider.KeyInfo(expires).PreviousKeyExpiresAt, "an expired rotation is not reported")

	redacted := provider.Redacted()
	assert.Empty(t, redacted.APIKey)
	assert.Empty(t, redacted.PreviousAPIKey)
	assert.NotNil(t, redacted.KeyInfo(1_800_000_000), "the redacted copy still describes the key")
}

func TestKnownContextWindow_ReportsOnlyFamiliesItCanPlace(t *testing.T) {
	t.Parallel()

	tokens, ok := aiprovider.KnownContextWindow("claude-sonnet-4-5")
	assert.True(t, ok)
	assert.Equal(t, 200_000, tokens)

	_, ok = aiprovider.KnownContextWindow("my-private-finetune")
	assert.False(t, ok)
	assert.Equal(t, aiprovider.DefaultContextWindow, aiprovider.ContextWindowFor("my-private-finetune"))
}

func TestChangeRules_NameALimitOrKeyChange(t *testing.T) {
	t.Parallel()

	before := validProvider()
	after := *before
	after.TimeoutSeconds = 120
	limits := ruleFor(t, "timeoutSeconds")
	assert.False(t, limits.Same(before, &after))

	rotated := *before
	rotated.APIKeyLastFour = "0000"
	key := ruleFor(t, "apiKey")
	assert.False(t, key.Same(before, &rotated))
	assert.True(t, key.Same(before, before))
}

func ruleFor(t *testing.T, field string) editchange.Rule[aiprovider.Provider] {
	t.Helper()

	for _, rule := range aiprovider.ChangeRules {
		if rule.Field == field {
			return rule
		}
	}
	t.Fatalf("no change rule for %s", field)

	return editchange.Rule[aiprovider.Provider]{}
}
