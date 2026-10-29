package nsconnector

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/accounting/nsbinding"
	"github.com/emoss08/trenova/internal/infrastructure/accounting/nsrefsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/shared/netsuite"
	"github.com/emoss08/trenova/shared/netsuiteerrors"
	"github.com/emoss08/trenova/shared/stringutils"
)

const maxCertificateSubject = 64

func (c *Connector) NormalizeAccountID(raw string) (string, error) {
	return netsuite.NormalizeAccountID(raw)
}

func (c *Connector) PrepareCertificate(
	subject string,
) (*accountingconnectorports.AccountingCertificate, error) {
	cert, err := netsuite.NewCertificate(netsuite.CertificateRequest{
		Subject: stringutils.TruncateBytes(
			nsrefsync.CleanText(subject, maxCertificateSubject),
			maxCertificateSubject,
		),
		Now:      c.Clock(),
		Validity: netsuite.MaxCertificateValidity,
	})
	if err != nil {
		return nil, err
	}
	return &accountingconnectorports.AccountingCertificate{
		PrivateKeyPEM:  cert.PrivateKeyPEM,
		CertificatePEM: cert.CertificatePEM,
		ExpiresAt:      cert.ExpiresAt,
	}, nil
}

func (c *Connector) TokenFromCertificate(
	ctx context.Context,
	grant *accountingconnectorports.AccountingCertificateGrant,
) (*accountingconnectorports.AccountingTokenGrant, error) {
	if c.oauth == nil {
		return nil, nsbinding.ErrNoRedirect
	}
	account, err := nsbinding.AccountOf(grant.RealmID)
	if err != nil {
		return nil, err
	}
	key, err := netsuite.ParsePrivateKey(grant.PrivateKeyPEM)
	if err != nil {
		return nil, err
	}
	token, err := c.oauth.TokenFromCertificate(ctx, netsuite.CertificateGrant{
		AccountID:     account,
		CertificateID: grant.CertificateID,
		PrivateKey:    key,
	})
	if err != nil {
		if netsuiteerrors.IsInvalidClient(err) || netsuiteerrors.IsInvalidGrant(err) {
			return nil, fmt.Errorf("%w: %w", nsbinding.ErrCertificateRevoked, err)
		}
		return nil, err
	}
	return &accountingconnectorports.AccountingTokenGrant{
		AccessToken:    token.AccessToken,
		AccessTokenTTL: token.ExpiresIn,
	}, nil
}
