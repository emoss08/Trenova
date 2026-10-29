package netsuite_test

import (
	"testing"

	"github.com/emoss08/trenova/shared/netsuite"
	"github.com/emoss08/trenova/shared/netsuiteerrors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccountHosts(t *testing.T) {
	t.Parallel()

	cases := []struct {
		account string
		host    string
		sandbox bool
	}{
		{account: "1234567", host: "1234567"},
		{account: "1234567_SB1", host: "1234567-sb1", sandbox: true},
		{account: "1234567_sb12", host: "1234567-sb12", sandbox: true},
		{account: "1234567_RP", host: "1234567-rp"},
		{account: "TSTDRV1234567", host: "tstdrv1234567"},
	}
	for _, tc := range cases {
		host, err := netsuite.AccountHost(tc.account)
		require.NoError(t, err, tc.account)
		assert.Equal(t, tc.host, host)

		api, err := netsuite.APIHost(tc.account)
		require.NoError(t, err)
		assert.Equal(t, tc.host+".suitetalk.api.netsuite.com", api)

		app, err := netsuite.AppHost(tc.account)
		require.NoError(t, err)
		assert.Equal(t, tc.host+".app.netsuite.com", app)

		assert.Equal(t, tc.sandbox, netsuite.IsSandboxAccount(tc.account), tc.account)
		assert.True(t, netsuite.IsAllowedHost(api))
		assert.True(t, netsuite.IsAllowedHost(app))
	}
}

func TestAccountIDValidation(t *testing.T) {
	t.Parallel()

	for _, valid := range []string{"1234567", "1234567_SB1", "1234567_RP", "TSTDRV99", "td123"} {
		assert.True(t, netsuite.ValidAccountID(valid), valid)
	}
	for _, invalid := range []string{
		"", " ", "1234567-SB1", "1234567_SB", "1234567_XX", "1234567_SB1_RP", "_SB1",
		"1234567.evil.com", "1234567/x", "1234567 ", "12 34",
	} {
		if invalid == "1234567 " {
			assert.True(t, netsuite.ValidAccountID(invalid), "surrounding space is trimmed")
			continue
		}
		assert.False(t, netsuite.ValidAccountID(invalid), invalid)
		_, err := netsuite.APIHost(invalid)
		require.ErrorIs(t, err, netsuiteerrors.ErrInvalidAccountID, invalid)
	}

	normalized, err := netsuite.NormalizeAccountID("tstdrv123_sb2")
	require.NoError(t, err)
	assert.Equal(t, "TSTDRV123_SB2", normalized)
}

func TestAllowedHosts(t *testing.T) {
	t.Parallel()

	hosts, err := netsuite.AllowedHosts(testAccount)
	require.NoError(t, err)
	assert.Equal(t, []string{
		"1234567-sb1.suitetalk.api.netsuite.com",
		"1234567-sb1.app.netsuite.com",
		"system.netsuite.com",
	}, hosts)

	assert.True(t, netsuite.IsAllowedHost("SYSTEM.netsuite.com"))
	for _, host := range []string{
		"evil.example", "netsuite.com", "suitetalk.api.netsuite.com", "a.b.suitetalk.api.netsuite.com",
		"1234567_sb1.app.netsuite.com", "1234567.suitetalk.api.netsuite.com.evil.example",
		"system.netsuite.com.evil.example",
	} {
		assert.False(t, netsuite.IsAllowedHost(host), host)
	}

	_, err = netsuite.AllowedHosts("bad id")
	require.ErrorIs(t, err, netsuiteerrors.ErrInvalidAccountID)
}

func TestTransactionLink(t *testing.T) {
	t.Parallel()

	assert.Equal(t,
		"https://1234567-sb1.app.netsuite.com/app/accounting/transactions/transaction.nl?id=1201",
		netsuite.TransactionLink(testAccount, "1201"))
	assert.Empty(t, netsuite.TransactionLink(testAccount, "12&x=1"))
	assert.Empty(t, netsuite.TransactionLink("bad id", "1201"))
}

func TestCompanyRefRoundTrips(t *testing.T) {
	t.Parallel()

	cases := []struct {
		raw string
		ref netsuite.CompanyRef
	}{
		{raw: "1234567", ref: netsuite.CompanyRef{AccountID: "1234567"}},
		{raw: "1234567-s3", ref: netsuite.CompanyRef{AccountID: "1234567", SubsidiaryID: "3"}},
		{
			raw: "1234567_SB1-s120",
			ref: netsuite.CompanyRef{AccountID: "1234567_SB1", SubsidiaryID: "120"},
		},
		{raw: "TSTDRV42_RP", ref: netsuite.CompanyRef{AccountID: "TSTDRV42_RP"}},
	}
	for _, tc := range cases {
		ref, err := netsuite.ParseCompanyRef(tc.raw)
		require.NoError(t, err, tc.raw)
		assert.Equal(t, tc.ref, ref)
		assert.Equal(t, tc.raw, ref.String())
		require.NoError(t, ref.Validate())
	}

	ref, err := netsuite.ParseCompanyRef(" 1234567_sb1-s7 ")
	require.NoError(t, err)
	assert.Equal(t, "1234567_SB1-s7", ref.String())
	assert.True(t, ref.Sandbox())
	assert.Equal(t, "1234567-sb1.suitetalk.api.netsuite.com", ref.APIHost())
	assert.Equal(t, "1234567-sb1.app.netsuite.com", ref.AppHost())

	production := netsuite.CompanyRef{AccountID: "1234567", SubsidiaryID: "1"}
	assert.False(t, production.Sandbox())
	assert.Equal(t, "1234567-s1", production.String())
}

func TestCompanyRefRejects(t *testing.T) {
	t.Parallel()

	for _, raw := range []string{
		"", "-s3", "1234567-", "1234567-s", "1234567-3", "1234567-S3", "1234567-s0",
		"1234567-s03", "1234567-s-3", "1234567-sx", "1234567-s3-s4", "1234567_SB-s3",
		"1234567-s1234567890123456789", "bad id-s3",
	} {
		_, err := netsuite.ParseCompanyRef(raw)
		require.ErrorIs(t, err, netsuiteerrors.ErrInvalidCompanyRef, raw)
	}

	_, err := netsuite.NewCompanyRef("1234567", "0")
	require.ErrorIs(t, err, netsuiteerrors.ErrInvalidSubsidiaryID)
	_, err = netsuite.NewCompanyRef("1234567-x", "")
	require.ErrorIs(t, err, netsuiteerrors.ErrInvalidAccountID)

	invalid := netsuite.CompanyRef{AccountID: "1234567", SubsidiaryID: "abc"}
	assert.Empty(t, invalid.String())
	assert.Empty(t, netsuite.CompanyRef{}.APIHost())
	require.Error(t, invalid.Validate())
}
