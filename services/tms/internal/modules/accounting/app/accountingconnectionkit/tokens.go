package accountingconnectionkit

import (
	"context"
	"errors"
	"regexp"
	"time"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingconnection"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingprovider"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectionports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/internal/modules/identity/app/planservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/platformplan"
	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const (
	accessTokenRefreshWindow = int64(5 * 60)
	AccessTokenField         = "access_token"
	RefreshTokenField        = "refresh_token"
	ClientKeyField           = "client_key"
)

var RealmIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,100}$`)

func (k *Kit) RequireIntegrations(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) error {
	return planservice.RequireCapability(
		ctx,
		k.plans,
		tenantInfo,
		platformplan.CapabilityIntegrations,
	)
}

func (k *Kit) RequestReferenceRefresh(
	ctx context.Context,
	conn *accountingconnection.AccountingConnection,
) {
	if k.refresher == nil {
		return
	}
	if err := k.refresher.RequestReferenceRefresh(ctx, pagination.TenantInfo{
		OrgID: conn.OrganizationID,
		BuID:  conn.BusinessUnitID,
	}, conn.ID); err != nil {
		k.l.Warn("could not start the reference data refresh after connecting",
			zap.String("connectionId", conn.ID.String()), zap.Error(err))
	}
}

func (k *Kit) Connect(
	conn *accountingconnection.AccountingConnection,
	userID pulid.ID,
	grant *accountingconnectorports.AccountingTokenGrant,
	facts *accountingconnection.CompanyFacts,
	now int64,
) error {
	sealed, err := k.Seal(conn, grant, now)
	if err != nil {
		return err
	}
	conn.Connect(userID, sealed, now)
	conn.ApplyCompanyFacts(facts)

	multiErr := errortypes.NewMultiError()
	conn.Validate(multiErr)
	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

func (k *Kit) RevokeStored(
	ctx context.Context,
	provider accountingconnectorports.AccountingProvider,
	conn *accountingconnection.AccountingConnection,
	refreshCiphertext string,
) {
	connector, err := k.ConnectorFor(ctx, provider, conn)
	if err != nil {
		k.l.Warn("could not resolve the app to revoke the authorization with", zap.Error(err))
		return
	}
	refreshToken, err := k.Open(conn, RefreshTokenField, refreshCiphertext)
	if err != nil {
		k.l.Warn("could not read refresh token to revoke it", zap.Error(err))
		return
	}
	k.RevokeQuietly(ctx, connector, refreshToken, conn.ExternalRealmID)
}

type TokenOutcome struct {
	Conn        *accountingconnection.AccountingConnection
	Connector   accountingconnectorports.AccountingConnector
	AccessToken string
	Failure     error
	Category    accountingsync.ErrorCategory
	Before      accountingconnection.ConnectionStatus
}

func (k *Kit) FreshAccessToken(
	ctx context.Context,
	provider accountingconnectorports.AccountingProvider,
	tenantInfo pagination.TenantInfo,
	connectionID pulid.ID,
	now int64,
) (*TokenOutcome, error) {
	outcome := &TokenOutcome{}
	err := k.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		conn, lockErr := k.connections.LockWithTokens(
			txCtx,
			accountingconnectionports.GetAccountingConnectionByIDRequest{
				TenantInfo: tenantInfo,
				ID:         connectionID,
			},
		)
		if lockErr != nil {
			return lockErr
		}
		outcome.Conn = conn
		outcome.Before = conn.Status
		if !conn.IsActive() {
			return nil
		}

		connector, bindErr := k.ConnectorFor(txCtx, provider, conn)
		if bindErr != nil {
			outcome.Failure = bindErr
			outcome.Category = accountingsync.ErrorCategoryConfiguration
			return k.RecordFailure(txCtx, conn, outcome.Category, bindErr, now)
		}
		outcome.Connector = connector

		if renewErr := k.renewAccessToken(ctx, txCtx, connector, conn, now); renewErr != nil {
			outcome.Failure = renewErr
			outcome.Category = connector.ClassifyError(renewErr)
			return k.RecordFailure(txCtx, conn, outcome.Category, renewErr, now)
		}

		if !conn.HasTokens() {
			outcome.Failure = errors.New("no stored authorization")
			outcome.Category = accountingsync.ErrorCategoryRevoked
			return k.RecordFailure(txCtx, conn, outcome.Category, outcome.Failure, now)
		}

		access, openErr := k.Open(conn, AccessTokenField, conn.AccessTokenCiphertext)
		if openErr != nil {
			outcome.Failure = openErr
			outcome.Category = accountingsync.ErrorCategoryConfiguration
			return k.RecordFailure(txCtx, conn, outcome.Category, errors.New(
				"the stored authorization could not be read; reconnect to replace it",
			), now)
		}
		outcome.AccessToken = access
		return nil
	})
	return outcome, err
}

func (k *Kit) renewAccessToken(
	ctx, txCtx context.Context,
	connector accountingconnectorports.AccountingConnector,
	conn *accountingconnection.AccountingConnection,
	now int64,
) error {
	if conn.UsesCertificate() {
		if !conn.HasCertificate() ||
			(conn.AccessTokenCiphertext != "" &&
				!conn.AccessTokenExpiresWithin(now, accessTokenRefreshWindow)) {
			return nil
		}
		return k.tokenFromStoredCertificate(ctx, txCtx, connector, conn, now)
	}
	if conn.HasTokens() && conn.AccessTokenExpiresWithin(now, accessTokenRefreshWindow) {
		return k.refresh(ctx, txCtx, connector, conn, now)
	}
	return nil
}

func (k *Kit) refresh(
	ctx, txCtx context.Context,
	connector accountingconnectorports.AccountingConnector,
	conn *accountingconnection.AccountingConnection,
	now int64,
) error {
	refreshToken, err := k.Open(conn, RefreshTokenField, conn.RefreshTokenCiphertext)
	if err != nil {
		return err
	}
	grant, err := connector.Refresh(ctx, &accountingconnectorports.AccountingRefreshRequest{
		RefreshToken: refreshToken,
		RealmID:      conn.ExternalRealmID,
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

func (k *Kit) RecordFailure(
	txCtx context.Context,
	conn *accountingconnection.AccountingConnection,
	category accountingsync.ErrorCategory,
	cause error,
	now int64,
) error {
	conn.RecordFailure(category, failureMessage(category, cause), now)
	if _, err := k.connections.Update(txCtx, conn); err != nil {
		return err
	}
	if category == accountingsync.ErrorCategoryRevoked {
		return k.connections.StoreTokens(txCtx, TokensOf(conn, now))
	}
	return nil
}

func (k *Kit) RevokeQuietly(
	ctx context.Context,
	connector accountingconnectorports.AccountingConnector,
	refreshToken, realmID string,
) {
	if refreshToken == "" {
		return
	}
	if err := connector.Revoke(ctx, &accountingconnectorports.AccountingRevokeRequest{
		Token:   refreshToken,
		RealmID: realmID,
	}); err != nil {
		k.l.Warn("could not revoke accounting authorization", zap.Error(err))
	}
}

func ConnectionAAD(
	conn *accountingconnection.AccountingConnection,
	field string,
) encryptionservice.AAD {
	return encryptionservice.AAD{
		Purpose:        encryptionservice.PurposeAccountingConnectionToken,
		OrganizationID: conn.OrganizationID,
		BusinessUnitID: conn.BusinessUnitID,
		ResourceID:     conn.ID.String() + ":" + field,
	}
}

func (k *Kit) Seal(
	conn *accountingconnection.AccountingConnection,
	grant *accountingconnectorports.AccountingTokenGrant,
	now int64,
) (accountingconnection.TokenGrant, error) {
	access, err := k.encryption.EncryptStringWithAAD(
		grant.AccessToken,
		ConnectionAAD(conn, AccessTokenField),
	)
	if err != nil {
		return accountingconnection.TokenGrant{}, err
	}
	sealed := accountingconnection.TokenGrant{
		AccessTokenCiphertext: access,
		AccessTokenExpiresAt:  now + int64(grant.AccessTokenTTL/time.Second),
	}
	if grant.RefreshToken == "" {
		sealed.RefreshTokenCiphertext = conn.RefreshTokenCiphertext
		sealed.RefreshTokenExpiresAt = conn.RefreshTokenExpiresAt
		return sealed, nil
	}

	refresh, err := k.encryption.EncryptStringWithAAD(
		grant.RefreshToken,
		ConnectionAAD(conn, RefreshTokenField),
	)
	if err != nil {
		return accountingconnection.TokenGrant{}, err
	}

	refreshLifetime := int64(grant.RefreshTokenTTL / time.Second)
	if refreshLifetime <= 0 {
		refreshLifetime = int64(
			accountingprovider.MustProfile(conn.IntegrationType).RefreshTokenLifetime / time.Second,
		)
	}
	sealed.RefreshTokenCiphertext = refresh
	sealed.RefreshTokenExpiresAt = now + refreshLifetime
	return sealed, nil
}

func (k *Kit) Open(
	conn *accountingconnection.AccountingConnection,
	field, ciphertext string,
) (string, error) {
	return k.encryption.DecryptStringWithAAD(ciphertext, ConnectionAAD(conn, field))
}

func TokensOf(
	conn *accountingconnection.AccountingConnection,
	now int64,
) accountingconnectionports.StoreAccountingTokensRequest {
	return accountingconnectionports.StoreAccountingTokensRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: conn.OrganizationID,
			BuID:  conn.BusinessUnitID,
		},
		ID:                     conn.ID,
		AccessTokenCiphertext:  conn.AccessTokenCiphertext,
		AccessTokenExpiresAt:   conn.AccessTokenExpiresAt,
		RefreshTokenCiphertext: conn.RefreshTokenCiphertext,
		RefreshTokenExpiresAt:  conn.RefreshTokenExpiresAt,
		ClientKeyCiphertext:    conn.ClientKeyCiphertext,
		RefreshedAt:            conn.LastRefreshedAt,
		At:                     now,
	}
}

func failureMessage(category accountingsync.ErrorCategory, cause error) string {
	switch category { //nolint:exhaustive // other categories describe themselves through the cause
	case accountingsync.ErrorCategoryRevoked:
		return "The authorization was revoked or has expired. Reconnect to resume."
	case accountingsync.ErrorCategoryRateLimited:
		return "The provider asked Trenova to slow down; calls resume automatically."
	case accountingsync.ErrorCategoryUnauthorized:
		return "The provider refused the stored authorization. Reconnect if this continues."
	case accountingsync.ErrorCategoryTransient:
		return "The provider did not answer in time; Trenova retries automatically."
	default:
		if cause == nil {
			return ""
		}
		return cause.Error()
	}
}
