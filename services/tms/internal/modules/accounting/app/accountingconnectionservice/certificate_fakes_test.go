package accountingconnectionservice

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectionports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
)

type fakeCertificates struct {
	mu   sync.Mutex
	rows map[string]*accountingsync.AccountingPendingCertificate
}

func newFakeCertificates() *fakeCertificates {
	return &fakeCertificates{rows: map[string]*accountingsync.AccountingPendingCertificate{}}
}

func certificateKey(req accountingconnectionports.GetAccountingConnectionRequest) string {
	return integrationKey(req.TenantInfo, req.IntegrationType)
}

func (f *fakeCertificates) GetByType(
	_ context.Context,
	req accountingconnectionports.GetAccountingConnectionRequest,
) (*accountingsync.AccountingPendingCertificate, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	row, ok := f.rows[certificateKey(req)]
	if !ok {
		return nil, errortypes.NewNotFoundError("Pending accounting certificate not found")
	}
	clone := *row
	return &clone, nil
}

func (f *fakeCertificates) Replace(
	_ context.Context,
	entity *accountingsync.AccountingPendingCertificate,
) (*accountingsync.AccountingPendingCertificate, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	clone := *entity
	f.rows[certificateKey(accountingconnectionports.GetAccountingConnectionRequest{
		TenantInfo:      pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID},
		IntegrationType: entity.IntegrationType,
	})] = &clone
	return entity, nil
}

func (f *fakeCertificates) Delete(
	_ context.Context,
	req accountingconnectionports.GetAccountingConnectionRequest,
) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.rows, certificateKey(req))
	return nil
}

func (f *fakeCertificates) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.rows)
}

const (
	testCertificatePEM = "-----BEGIN CERTIFICATE-----\nfake\n-----END CERTIFICATE-----\n"
	testPrivateKeyPEM  = "-----BEGIN PRIVATE KEY-----\nfake-key\n-----END PRIVATE KEY-----\n"
)

type certificateCall struct {
	realmID       string
	certificateID string
	privateKey    string
}

type fakeClientCredentials struct {
	mu             sync.Mutex
	subjects       []string
	preparedKeys   []string
	assertions     []certificateCall
	assertionErrs  []error
	assertionGrant *accountingconnectorports.AccountingTokenGrant
	expiresAt      time.Time
}

func (f *fakeConnector) NormalizeAccountID(raw string) (string, error) {
	normalized := strings.ToUpper(strings.TrimSpace(raw))
	if normalized == "" || strings.ContainsAny(normalized, " /.") {
		return "", errProvider
	}
	return normalized, nil
}

func (f *fakeConnector) PrepareCertificate(
	subject string,
) (*accountingconnectorports.AccountingCertificate, error) {
	cc := f.clientCredentials()
	cc.mu.Lock()
	defer cc.mu.Unlock()
	cc.subjects = append(cc.subjects, subject)
	key := testPrivateKeyPEM + subject
	cc.preparedKeys = append(cc.preparedKeys, key)
	return &accountingconnectorports.AccountingCertificate{
		PrivateKeyPEM:  []byte(key),
		CertificatePEM: []byte(testCertificatePEM),
		ExpiresAt:      cc.expiresAt,
	}, nil
}

func (f *fakeConnector) TokenFromCertificate(
	_ context.Context,
	grant *accountingconnectorports.AccountingCertificateGrant,
) (*accountingconnectorports.AccountingTokenGrant, error) {
	cc := f.clientCredentials()
	cc.mu.Lock()
	defer cc.mu.Unlock()
	cc.assertions = append(cc.assertions, certificateCall{
		realmID:       grant.RealmID,
		certificateID: grant.CertificateID,
		privateKey:    string(grant.PrivateKeyPEM),
	})
	if len(cc.assertionErrs) > 0 {
		err := cc.assertionErrs[0]
		cc.assertionErrs = cc.assertionErrs[1:]
		if err != nil {
			return nil, err
		}
	}
	clone := *cc.assertionGrant
	return &clone, nil
}

func (f *fakeConnector) clientCredentials() *fakeClientCredentials {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.cc == nil {
		f.cc = &fakeClientCredentials{
			assertionGrant: &accountingconnectorports.AccountingTokenGrant{
				AccessToken:    "cert-access",
				AccessTokenTTL: time.Hour,
			},
			expiresAt: time.Now().Add(2 * 365 * 24 * time.Hour).UTC().Truncate(time.Second),
		}
	}
	return f.cc
}
