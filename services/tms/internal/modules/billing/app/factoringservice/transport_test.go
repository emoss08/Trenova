package factoringservice

import (
	"errors"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/factoringcompany"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newTransportHarness() (*TransportService, *fakeTransports, *fakeSender, pagination.TenantInfo, pulid.ID) {
	repo := &fakeTransports{}
	sftp := &fakeSender{method: factoring.TransportMethodSftp}
	factorID := pulid.MustNew("fac_")
	svc := NewTransportService(TransportParams{
		Repo:       repo,
		Factors:    &fakeFactors{factor: &factoringcompany.FactoringCompany{ID: factorID, Name: "Lone Star"}},
		Cipher:     fakeCipher{},
		Transports: []factoringports.FactoringTransport{sftp},
		Logger:     zap.NewNop(),
	}).(*TransportService)
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	return svc, repo, sftp, tenant, factorID
}

func sftpInput(secret *string) factoringports.TransportInput {
	return factoringports.TransportInput{
		Method:           factoring.TransportMethodSftp,
		SftpHost:         "sftp.lonestar.test",
		SftpUsername:     "carrier",
		SftpAuthMethod:   factoring.SftpAuthPassword,
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

	saved, err := svc.Save(t.Context(), &factoringports.SaveTransportRequest{
		TenantInfo:         tenant,
		FactoringCompanyID: factorID,
		Input:              sftpInput(&secret),
	}, actor)
	require.NoError(t, err)
	assert.Equal(t, 22, saved.SftpPort)
	assert.True(t, strings.HasPrefix(repo.entity.SftpSecret, "sealed:"+saved.ID.String()+":sftp_secret:"))
	assert.NotContains(t, redacted(saved).SftpSecret, "hunter2")

	keep := sftpInput(nil)
	version := saved.Version
	kept, err := svc.Save(t.Context(), &factoringports.SaveTransportRequest{
		TenantInfo:         tenant,
		FactoringCompanyID: factorID,
		Version:            &version,
		Input:              keep,
	}, actor)
	require.NoError(t, err)
	assert.Equal(t, repo.entity.SftpSecret, kept.SftpSecret)

	stale := version
	_, err = svc.Save(t.Context(), &factoringports.SaveTransportRequest{
		TenantInfo:         tenant,
		FactoringCompanyID: factorID,
		Version:            &stale,
		Input:              keep,
	}, actor)
	require.Error(t, err)
	assert.True(t, errortypes.IsConflictError(err))

	switched := sftpInput(nil)
	switched.SftpAuthMethod = factoring.SftpAuthPrivateKey
	current := kept.Version
	_, err = svc.Save(t.Context(), &factoringports.SaveTransportRequest{
		TenantInfo:         tenant,
		FactoringCompanyID: factorID,
		Version:            &current,
		Input:              switched,
	}, actor)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SFTP password or private key")
}

func TestTestTransportRecordsTheOutcome(t *testing.T) {
	t.Parallel()
	svc, repo, sftp, tenant, factorID := newTransportHarness()
	actor := &permissionports.RequestActor{UserID: pulid.MustNew("usr_")}
	secret := "hunter2"
	_, err := svc.Save(t.Context(), &factoringports.SaveTransportRequest{
		TenantInfo:         tenant,
		FactoringCompanyID: factorID,
		Input:              sftpInput(&secret),
	}, actor)
	require.NoError(t, err)

	result, err := svc.Test(t.Context(), &factoringports.TestTransportRequest{
		TenantInfo:         tenant,
		FactoringCompanyID: factorID,
	}, actor)
	require.NoError(t, err)
	assert.True(t, result.Passed)
	require.NotNil(t, repo.tested)
	assert.Empty(t, repo.tested.LastTestError)

	sftp.testErr = errors.New("permission denied")
	result, err = svc.Test(t.Context(), &factoringports.TestTransportRequest{
		TenantInfo:         tenant,
		FactoringCompanyID: factorID,
	}, actor)
	require.NoError(t, err)
	assert.False(t, result.Passed)
	assert.Equal(t, "permission denied", repo.tested.LastTestError)
}
