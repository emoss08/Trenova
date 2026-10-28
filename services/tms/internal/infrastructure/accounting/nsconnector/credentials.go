package nsconnector

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/shared/netsuite"
	"github.com/emoss08/trenova/shared/stringutils"
)

const maxCertificateSubject = 64

var errCertificateRevoked = errors.New(
	"netsuite refused the certificate: the client credentials mapping is missing or was " +
		"cleared, for example by a sandbox refresh",
)

func (c *Connector) NormalizeAccountID(raw string) (string, error) {
	return netsuite.NormalizeAccountID(raw)
}

func (c *Connector) PrepareCertificate(
	subject string,
) (*accountingconnectorports.AccountingCertificate, error) {
	cert, err := netsuite.NewCertificate(netsuite.CertificateRequest{
		Subject: stringutils.TruncateBytes(
			cleanText(subject, maxCertificateSubject),
			maxCertificateSubject,
		),
		Now:      c.now(),
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
		return nil, errNoRedirect
	}
	account, err := accountOf(grant.RealmID)
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
		if netsuite.IsInvalidClient(err) || netsuite.IsInvalidGrant(err) {
			return nil, fmt.Errorf("%w: %w", errCertificateRevoked, err)
		}
		return nil, err
	}
	return &accountingconnectorports.AccountingTokenGrant{
		AccessToken:    token.AccessToken,
		AccessTokenTTL: token.ExpiresIn,
	}, nil
}
