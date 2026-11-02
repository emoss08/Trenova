package authstore

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/identity/domain/session"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const sealKeyLabel = "trenova session seal v1"

type sealedFields struct {
	ID                    pulid.ID   `json:"id"`
	SecretHash            string     `json:"secretHash"`
	PreviousSecretHash    string     `json:"previousSecretHash"`
	PreviousSecretUntil   int64      `json:"previousSecretUntil"`
	SessionEpoch          int64      `json:"sessionEpoch"`
	UserID                pulid.ID   `json:"userId"`
	BusinessUnitID        pulid.ID   `json:"businessUnitId"`
	OrganizationID        pulid.ID   `json:"organizationId"`
	ActiveRoleIDs         []pulid.ID `json:"activeRoleIds"`
	AuthProvider          string     `json:"authProvider"`
	ExternalIdentityID    string     `json:"externalIdentityId"`
	ExternalSubject       string     `json:"externalSubject"`
	AuthenticatorAAL      int        `json:"authenticatorAal"`
	FederationFAL         int        `json:"federationFal"`
	MFAAuthenticatedAt    int64      `json:"mfaAuthenticatedAt"`
	LastReauthenticatedAt int64      `json:"lastReauthenticatedAt"`
	RiskDecision          string     `json:"riskDecision"`
	RiskDecisionID        pulid.ID   `json:"riskDecisionId"`
	IsPortalUser          bool       `json:"isPortalUser"`
	IssuedBySignIn        bool       `json:"issuedBySignIn"`
	MustChangePassword    bool       `json:"mustChangePassword"`
	ExpiresAt             int64      `json:"expiresAt"`
	CreatedAt             int64      `json:"createdAt"`
}

func (r *sessionRepository) sealOf(sess *session.Session) (string, error) {
	roles := sess.ActiveRoleIDs
	if roles == nil {
		roles = []pulid.ID{}
	}
	payload, err := sonic.Marshal(sealedFields{
		ID:                    sess.ID,
		SecretHash:            sess.SecretHash,
		PreviousSecretHash:    sess.PreviousSecretHash,
		PreviousSecretUntil:   sess.PreviousSecretUntil,
		SessionEpoch:          sess.SessionEpoch,
		UserID:                sess.UserID,
		BusinessUnitID:        sess.BusinessUnitID,
		OrganizationID:        sess.OrganizationID,
		ActiveRoleIDs:         roles,
		AuthProvider:          sess.AuthProvider,
		ExternalIdentityID:    sess.ExternalIdentityID,
		ExternalSubject:       sess.ExternalSubject,
		AuthenticatorAAL:      sess.AuthenticatorAAL,
		FederationFAL:         sess.FederationFAL,
		MFAAuthenticatedAt:    sess.MFAAuthenticatedAt,
		LastReauthenticatedAt: sess.LastReauthenticatedAt,
		RiskDecision:          sess.RiskDecision,
		RiskDecisionID:        sess.RiskDecisionID,
		IsPortalUser:          sess.IsPortalUser,
		IssuedBySignIn:        sess.IssuedBySignIn,
		MustChangePassword:    sess.MustChangePassword,
		ExpiresAt:             sess.ExpiresAt,
		CreatedAt:             sess.CreatedAt,
	})
	if err != nil {
		return "", err
	}

	mac := hmac.New(sha256.New, r.sealKey)
	_, _ = mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil)), nil
}

func (r *sessionRepository) seal(sess *session.Session) error {
	value, err := r.sealOf(sess)
	if err != nil {
		return err
	}
	sess.Seal = value
	return nil
}

func (r *sessionRepository) sealHolds(ctx context.Context, sess *session.Session) bool {
	want, err := r.sealOf(sess)
	if err == nil && sess.Seal != "" && hmac.Equal([]byte(want), []byte(sess.Seal)) {
		return true
	}

	r.l.Error(
		"discarded a stored session whose seal does not match its fields",
		zap.String("sessionID", sess.ID.String()),
		zap.String("userID", sess.UserID.String()),
		zap.Error(err),
	)
	r.discard(ctx, sess)
	return false
}
