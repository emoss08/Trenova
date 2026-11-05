package factoringtransportservice

import (
	"errors"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/app/factoringservicetest"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringsubmission"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringsubmissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/factoringcompany"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newTransportHarness() (*TransportService, *factoringservicetest.FakeTransports, *factoringservicetest.FakeSender, pagination.TenantInfo, pulid.ID) {
	repo := &factoringservicetest.FakeTransports{}
	sftp := &factoringservicetest.FakeSender{
		TransportMethod: factoringsubmission.TransportMethodSftp,
	}
	factorID := pulid.MustNew("fac_")
	svc := NewTransportService(TransportParams{
		Repo:       repo,
		Factors:    &factoringservicetest.FakeFactors{Factor: &factoringcompany.FactoringCompany{ID: factorID, Name: "Lone Star"}},
		Cipher:     factoringservicetest.FakeCipher{},
		Transports: []factoringsubmissionports.FactoringTransport{sftp},
		Logger:     zap.NewNop(),
	}).(*TransportService)
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	return svc, repo, sftp, tenant, factorID
}

func sftpInput(secret *string) factoringsubmissionports.TransportInput {
	return factoringsubmissionports.TransportInput{
		Method:           factoringsubmission.TransportMethodSftp,
		SftpHost:         "sftp.lonestar.test",
		SftpUsername:     "carrier",
		SftpAuthMethod:   factoringsubmission.SftpAuthPassword,
		SftpSecret:       secret,
		SftpKnownHostKey: "ssh-ed25519 AAAA",
		SftpDirectory:    "/in",
	}
}

func TestSaveTransportSealsTheSecretToTheRow(t *testing.T) {
	t.Parallel()
	svc, repo, _, tenant, factorID := newTransportHarness()
	actor := &permissionports.RequestActor{UserID: pulid.MustNew("usr_")}
	secret := "hunter2"

	saved, err := svc.Save(t.Context(), &factoringsubmissionports.SaveTransportRequest{
		TenantInfo:         tenant,
		FactoringCompanyID: factorID,
		Input:              sftpInput(&secret),
	}, actor)
	require.NoError(t, err)
	assert.Equal(t, 22, saved.SftpPort)
	assert.True(
		t,
		strings.HasPrefix(repo.Entity.SftpSecret, "sealed:"+saved.ID.String()+":sftp_secret:"),
	)
	assert.NotContains(t, redacted(saved).SftpSecret, "hunter2")

	keep := sftpInput(nil)
	version := saved.Version
	kept, err := svc.Save(t.Context(), &factoringsubmissionports.SaveTransportRequest{
		TenantInfo:         tenant,
		FactoringCompanyID: factorID,
		Version:            &version,
		Input:              keep,
	}, actor)
	require.NoError(t, err)
	assert.Equal(t, repo.Entity.SftpSecret, kept.SftpSecret)

	stale := version
	_, err = svc.Save(t.Context(), &factoringsubmissionports.SaveTransportRequest{
		TenantInfo:         tenant,
		FactoringCompanyID: factorID,
		Version:            &stale,
		Input:              keep,
	}, actor)
	require.Error(t, err)
	assert.True(t, errortypes.IsConflictError(err))

	switched := sftpInput(nil)
	switched.SftpAuthMethod = factoringsubmission.SftpAuthPrivateKey
	current := kept.Version
	_, err = svc.Save(t.Context(), &factoringsubmissionports.SaveTransportRequest{
		TenantInfo:         tenant,
		FactoringCompanyID: factorID,
		Version:            &current,
		Input:              switched,
	}, actor)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SFTP password or private key")
}

func TestSaveTransportRefusesTheSavedSecretWhenTheSFTPTargetChanges(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		change func(*factoringsubmissionports.TransportInput)
	}{
		{
			name: "host",
			change: func(in *factoringsubmissionports.TransportInput) {
				in.SftpHost = "sftp.attacker.test"
			},
		},
		{
			name:   "port",
			change: func(in *factoringsubmissionports.TransportInput) { in.SftpPort = 2222 },
		},
		{
			name: "known host key",
			change: func(in *factoringsubmissionports.TransportInput) {
				in.SftpKnownHostKey = "ssh-ed25519 BBBB"
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			svc, _, _, tenant, factorID := newTransportHarness()
			actor := &permissionports.RequestActor{UserID: pulid.MustNew("usr_")}
			secret := "hunter2"
			saved, err := svc.Save(t.Context(), &factoringsubmissionports.SaveTransportRequest{
				TenantInfo:         tenant,
				FactoringCompanyID: factorID,
				Input:              sftpInput(&secret),
			}, actor)
			require.NoError(t, err)

			moved := sftpInput(nil)
			tc.change(&moved)
			version := saved.Version
			_, err = svc.Save(t.Context(), &factoringsubmissionports.SaveTransportRequest{
				TenantInfo:         tenant,
				FactoringCompanyID: factorID,
				Version:            &version,
				Input:              moved,
			}, actor)

			require.Error(t, err)
			assert.Contains(t, err.Error(), "SFTP password or private key")
		})
	}
}

func TestSaveTransportKeepsTheSecretWhenOnlyHostWhitespaceDiffers(t *testing.T) {
	t.Parallel()
	svc, repo, _, tenant, factorID := newTransportHarness()
	actor := &permissionports.RequestActor{UserID: pulid.MustNew("usr_")}
	secret := "hunter2"
	saved, err := svc.Save(t.Context(), &factoringsubmissionports.SaveTransportRequest{
		TenantInfo:         tenant,
		FactoringCompanyID: factorID,
		Input:              sftpInput(&secret),
	}, actor)
	require.NoError(t, err)
	sealed := repo.Entity.SftpSecret

	same := sftpInput(nil)
	same.SftpHost = "  sftp.lonestar.test "
	version := saved.Version
	kept, err := svc.Save(t.Context(), &factoringsubmissionports.SaveTransportRequest{
		TenantInfo:         tenant,
		FactoringCompanyID: factorID,
		Version:            &version,
		Input:              same,
	}, actor)

	require.NoError(t, err)
	assert.Equal(t, sealed, kept.SftpSecret)
}

func TestSaveTransportSealsANewSecretForANewSFTPTarget(t *testing.T) {
	t.Parallel()
	svc, _, _, tenant, factorID := newTransportHarness()
	actor := &permissionports.RequestActor{UserID: pulid.MustNew("usr_")}
	secret := "hunter2"
	saved, err := svc.Save(t.Context(), &factoringsubmissionports.SaveTransportRequest{
		TenantInfo:         tenant,
		FactoringCompanyID: factorID,
		Input:              sftpInput(&secret),
	}, actor)
	require.NoError(t, err)

	replacement := "correct-horse"
	moved := sftpInput(&replacement)
	moved.SftpHost = "sftp.moved.test"
	version := saved.Version
	updated, err := svc.Save(t.Context(), &factoringsubmissionports.SaveTransportRequest{
		TenantInfo:         tenant,
		FactoringCompanyID: factorID,
		Version:            &version,
		Input:              moved,
	}, actor)

	require.NoError(t, err)
	assert.True(t, updated.SecretSet())
}

func TestTestTransportRecordsTheOutcome(t *testing.T) {
	t.Parallel()
	svc, repo, sftp, tenant, factorID := newTransportHarness()
	actor := &permissionports.RequestActor{UserID: pulid.MustNew("usr_")}
	secret := "hunter2"
	_, err := svc.Save(t.Context(), &factoringsubmissionports.SaveTransportRequest{
		TenantInfo:         tenant,
		FactoringCompanyID: factorID,
		Input:              sftpInput(&secret),
	}, actor)
	require.NoError(t, err)

	result, err := svc.Test(t.Context(), &factoringsubmissionports.TestTransportRequest{
		TenantInfo:         tenant,
		FactoringCompanyID: factorID,
	}, actor)
	require.NoError(t, err)
	assert.True(t, result.Passed)
	require.NotNil(t, repo.Tested)
	assert.Empty(t, repo.Tested.LastTestError)

	sftp.TestErr = errors.New("permission denied")
	result, err = svc.Test(t.Context(), &factoringsubmissionports.TestTransportRequest{
		TenantInfo:         tenant,
		FactoringCompanyID: factorID,
	}, actor)
	require.NoError(t, err)
	assert.False(t, result.Passed)
	assert.Equal(t, "permission denied", repo.Tested.LastTestError)
}
