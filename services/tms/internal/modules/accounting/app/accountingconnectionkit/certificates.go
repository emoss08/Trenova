package accountingconnectionkit

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingconnection"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingprovider"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectionports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
)

type HeldCertificate struct {
	CertificateID string
	PrivateKeyPEM string
	ExpiresAt     int64
}

func ErrCertificatesUnsupported(provider string) error {
	return errortypes.NewBusinessError(
		"{0} connections cannot be made with a certificate. Sign in instead.",
		provider,
	)
}

func errNoPendingCertificate(provider string) error {
	return errortypes.NewBusinessError(
		"There is no {0} certificate waiting to be connected. Generate a certificate, upload it to {0}, then try again.",
		provider,
	)
}

func PendingKeyAAD(
	pending *accountingconnection.AccountingPendingCertificate,
) encryptionservice.AAD {
	return encryptionservice.AAD{
		Purpose:        encryptionservice.PurposeAccountingClientKey,
		OrganizationID: pending.OrganizationID,
		BusinessUnitID: pending.BusinessUnitID,
		ResourceID:     pending.ID.String(),
	}
}

func ClientCredentialsOf(
	connector accountingconnectorports.AccountingConnector,
	provider string,
) (accountingconnectorports.AccountingClientCredentialsConnector, error) {
	cc, ok := connector.(accountingconnectorports.AccountingClientCredentialsConnector)
	if !ok {
		return nil, ErrCertificatesUnsupported(provider)
	}
	return cc, nil
}

func (k *Kit) HeldPendingCertificate(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	typ integration.Type,
	certificateID string,
) (*HeldCertificate, error) {
	name := accountingprovider.ProviderName(typ)
	pending, err := k.certificates.GetByType(
		ctx,
		accountingconnectionports.GetAccountingConnectionRequest{
			TenantInfo:      tenantInfo,
			IntegrationType: typ,
		},
	)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, errNoPendingCertificate(name)
		}
		return nil, err
	}
	if pending.ExpiresAt <= timeutils.NowUnix() {
		return nil, errortypes.NewBusinessError(
			"The {0} certificate waiting to be connected has expired. Generate a new one.",
			name,
		)
	}
	key, err := k.encryption.DecryptStringWithAAD(
		pending.PrivateKeyCiphertext,
		PendingKeyAAD(pending),
	)
	if err != nil {
		return nil, errortypes.NewBusinessError(
			"The {0} certificate's key could not be read. Generate a new certificate.",
			name,
		).WithInternal(err)
	}
	return &HeldCertificate{
		CertificateID: certificateID,
		PrivateKeyPEM: key,
		ExpiresAt:     pending.ExpiresAt,
	}, nil
}

func (k *Kit) UseCertificate(
	conn *accountingconnection.AccountingConnection,
	held *HeldCertificate,
) error {
	sealed, err := k.encryption.EncryptStringWithAAD(
		held.PrivateKeyPEM,
		ConnectionAAD(conn, ClientKeyField),
	)
	if err != nil {
		return err
	}
	conn.UseCertificate(accountingconnection.ClientCertificate{
		KeyCiphertext: sealed,
		CertificateID: held.CertificateID,
		ExpiresAt:     held.ExpiresAt,
	})
	return nil
}

func (k *Kit) tokenFromStoredCertificate(
	ctx, txCtx context.Context,
	connector accountingconnectorports.AccountingConnector,
	conn *accountingconnection.AccountingConnection,
	now int64,
) error {
	cc, err := ClientCredentialsOf(connector, accountingprovider.ProviderName(conn.IntegrationType))
	if err != nil {
		return err
	}
	key, err := k.Open(conn, ClientKeyField, conn.ClientKeyCiphertext)
	if err != nil {
		return err
	}
	grant, err := cc.TokenFromCertificate(ctx, &accountingconnectorports.AccountingCertificateGrant{
		RealmID:       conn.ExternalRealmID,
		CertificateID: conn.ClientCertificateID,
		PrivateKeyPEM: []byte(key),
	})
	if err != nil {
		return err
	}
	sealed, err := k.Seal(conn, grant, now)
	if err != nil {
		return err
	}
	conn.ApplyGrant(sealed, now)
	return k.connections.StoreTokens(txCtx, TokensOf(conn, now))
}

func (k *Kit) ApplyConnectMethod(
	conn *accountingconnection.AccountingConnection,
	certificate *HeldCertificate,
) error {
	if certificate == nil {
		conn.UseAuthorization()
		return nil
	}
	return k.UseCertificate(conn, certificate)
}

func (k *Kit) ConsumePendingCertificate(
	txCtx context.Context,
	conn *accountingconnection.AccountingConnection,
	certificate *HeldCertificate,
) error {
	if certificate == nil || k.certificates == nil {
		return nil
	}
	return k.certificates.Delete(txCtx, accountingconnectionports.GetAccountingConnectionRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: conn.OrganizationID,
			BuID:  conn.BusinessUnitID,
		},
		IntegrationType: conn.IntegrationType,
	})
}

func (k *Kit) PendingCertificate(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	typ integration.Type,
) (*accountingconnectionports.AccountingPreparedCertificate, error) {
	if k.certificates == nil || !accountingprovider.MustProfile(typ).ClientCredentials {
		return nil, nil //nolint:nilnil // no certificate is waiting
	}
	pending, err := k.certificates.GetByType(
		ctx,
		accountingconnectionports.GetAccountingConnectionRequest{
			TenantInfo:      tenantInfo,
			IntegrationType: typ,
		},
	)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, nil //nolint:nilnil // no certificate is waiting
		}
		return nil, err
	}
	return PreparedCertificateOf(pending), nil
}

func PreparedCertificateOf(
	pending *accountingconnection.AccountingPendingCertificate,
) *accountingconnectionports.AccountingPreparedCertificate {
	return &accountingconnectionports.AccountingPreparedCertificate{
		IntegrationType: pending.IntegrationType,
		CertificatePEM:  pending.CertificatePEM,
		ExpiresAt:       pending.ExpiresAt,
		PreparedAt:      pending.PreparedAt,
	}
}
