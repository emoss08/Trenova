package tenant

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
	"golang.org/x/crypto/bcrypt"
)

var _ bun.BeforeAppendModelHook = (*APIToken)(nil)

const (
	// TokenLength is the length of the generated token
	TokenLength = 32
	// TokenPrefixLength is the length of the token prefix stored in plain text
	TokenPrefixLength = 8
)

// APITokenScope represents the permissions granted to an API token
type APITokenScope string

const (
	// Read-only scopes
	ScopeReadShipments  APITokenScope = "shipments:read"
	ScopeReadCustomers  APITokenScope = "customers:read"
	ScopeReadEquipment  APITokenScope = "equipment:read"
	ScopeReadDrivers    APITokenScope = "drivers:read"
	ScopeReadInvoices   APITokenScope = "invoices:read"
	ScopeReadReports    APITokenScope = "reports:read"
	ScopeReadDocuments  APITokenScope = "documents:read"
	ScopeReadCompliance APITokenScope = "compliance:read"

	// Write scopes
	ScopeWriteShipments APITokenScope = "shipments:write"
	ScopeWriteCustomers APITokenScope = "customers:write"
	ScopeWriteEquipment APITokenScope = "equipment:write"
	ScopeWriteDrivers   APITokenScope = "drivers:write"
	ScopeWriteInvoices  APITokenScope = "invoices:write"
	ScopeWriteDocuments APITokenScope = "documents:write"

	// Admin scopes
	ScopeAdminUsers        APITokenScope = "admin:users"
	ScopeAdminOrganization APITokenScope = "admin:organization"
	ScopeAdminBilling      APITokenScope = "admin:billing"

	// Special scopes
	ScopeFullAccess APITokenScope = "full_access"
	ScopeWebhooks   APITokenScope = "webhooks:manage"
)

// APIToken represents an API token for authentication
type APIToken struct {
	bun.BaseModel `bun:"table:api_tokens,alias:at" json:"-"`

	ID             pulid.ID           `json:"id"             bun:"id,pk,type:VARCHAR(100)"`
	UserID         pulid.ID           `json:"userId"         bun:"user_id,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID           `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID           `json:"organizationId" bun:"organization_id,type:VARCHAR(100),notnull"`
	Status         domaintypes.Status `json:"status"         bun:"status,type:status_enum,notnull,default:'Active'"`
	Name           string             `json:"name"           bun:"name,type:VARCHAR(255),notnull"`
	Description    string             `json:"description"    bun:"description,type:TEXT"`
	TokenHash      string             `json:"-"              bun:"token_hash,type:VARCHAR(255),notnull"`  // Hashed token
	TokenPrefix    string             `json:"tokenPrefix"    bun:"token_prefix,type:VARCHAR(10),notnull"` // First 8 chars for identification
	Scopes         []APITokenScope    `json:"scopes"         bun:"scopes,type:TEXT[],array"`
	LastUsedAt     *int64             `json:"lastUsedAt"     bun:"last_used_at,nullzero"`
	LastUsedIP     string             `json:"lastUsedIp"     bun:"last_used_ip,type:VARCHAR(45)"`
	ExpiresAt      *int64             `json:"expiresAt"      bun:"expires_at,nullzero"`
	Version        int64              `json:"version"        bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt      int64              `json:"createdAt"      bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64              `json:"updatedAt"      bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	User         *User         `json:"user,omitempty"         bun:"rel:belongs-to,join:user_id=id"`
	BusinessUnit *BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`

	// Transient field - only populated when creating a new token
	PlainToken string `json:"token,omitempty" bun:"-"`
}

// NewAPITokenRequest contains the data needed to create a new API token
type NewAPITokenRequest struct {
	UserID         pulid.ID
	BusinessUnitID pulid.ID
	OrganizationID pulid.ID
	Name           string
	Description    string
	Scopes         []APITokenScope
	ExpiresAt      *int64 // Optional expiration timestamp
}

// NewAPIToken creates a new API token
func NewAPIToken(req NewAPITokenRequest) (*APIToken, error) {
	// Generate random token
	tokenBytes := make([]byte, TokenLength)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	// Create base64 encoded token
	plainToken := base64.URLEncoding.EncodeToString(tokenBytes)
	plainToken = strings.TrimRight(plainToken, "=") // Remove padding

	// Extract prefix for identification
	tokenPrefix := plainToken[:TokenPrefixLength]

	// Hash the token
	hashedToken, err := bcrypt.GenerateFromPassword([]byte(plainToken), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash token: %w", err)
	}

	now := utils.NowUnix()

	token := &APIToken{
		ID:             pulid.MustNew("tok_"),
		UserID:         req.UserID,
		BusinessUnitID: req.BusinessUnitID,
		OrganizationID: req.OrganizationID,
		Status:         domaintypes.StatusActive,
		Name:           req.Name,
		Description:    req.Description,
		TokenHash:      string(hashedToken),
		TokenPrefix:    tokenPrefix,
		Scopes:         req.Scopes,
		ExpiresAt:      req.ExpiresAt,
		PlainToken:     plainToken, // This will be returned only once
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	// Validate the token
	if err := token.Validate(); err != nil {
		return nil, err
	}

	return token, nil
}

// Validate validates the API token
func (t *APIToken) Validate() error {
	multiErr := &errortypes.MultiError{}

	err := validation.ValidateStruct(t,
		validation.Field(
			&t.Name,
			validation.Required.Error("Token name is required"),
			validation.Length(1, 255).Error("Token name must be between 1 and 255 characters"),
		),
		validation.Field(
			&t.UserID,
			validation.Required.Error("User ID is required"),
		),
		validation.Field(
			&t.BusinessUnitID,
			validation.Required.Error("Business Unit ID is required"),
		),
		validation.Field(
			&t.OrganizationID,
			validation.Required.Error("Organization ID is required"),
		),
		validation.Field(
			&t.Scopes,
			validation.Required.Error("At least one scope is required"),
			validation.Length(1, 100).Error("Too many scopes specified"),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}

	// Custom validation for expiration
	if t.ExpiresAt != nil && *t.ExpiresAt <= utils.NowUnix() {
		multiErr.Add("expiresAt", errortypes.ErrInvalid, "Expiration time must be in the future")
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

// VerifyToken verifies a plain text token against the hashed token
func (t *APIToken) VerifyToken(plainToken string) error {
	// Check if token starts with the correct prefix
	if !strings.HasPrefix(plainToken, t.TokenPrefix) {
		return errortypes.NewAuthenticationError("Invalid token")
	}

	// Verify the token hash
	if err := bcrypt.CompareHashAndPassword([]byte(t.TokenHash), []byte(plainToken)); err != nil {
		return errortypes.NewAuthenticationError("Invalid token")
	}

	// Check if token is active
	if !t.IsActive() {
		return errortypes.NewAuthenticationError("Token is not active")
	}

	// Check if token is expired
	if t.IsExpired() {
		return errortypes.NewAuthenticationError("Token has expired")
	}

	return nil
}

// IsActive returns true if the token is active
func (t *APIToken) IsActive() bool {
	return t.Status == domaintypes.StatusActive
}

// IsExpired returns true if the token has expired
func (t *APIToken) IsExpired() bool {
	if t.ExpiresAt == nil {
		return false // No expiration set
	}
	return *t.ExpiresAt < utils.NowUnix()
}

// HasScope checks if the token has a specific scope
func (t *APIToken) HasScope(scope APITokenScope) bool {
	// Check for full access
	return slices.ContainsFunc(t.Scopes, func(s APITokenScope) bool {
		return s == ScopeFullAccess || s == scope
	})
}

// HasAnyScope checks if the token has any of the provided scopes
func (t *APIToken) HasAnyScope(scopes ...APITokenScope) bool {
	return slices.ContainsFunc(scopes, func(scope APITokenScope) bool {
		return t.HasScope(scope)
	})
}

// HasAllScopes checks if the token has all of the provided scopes
func (t *APIToken) HasAllScopes(scopes ...APITokenScope) bool {
	for _, scope := range scopes {
		if !t.HasScope(scope) {
			return false
		}
	}
	return true
}

// UpdateLastUsed updates the last used timestamp and IP
func (t *APIToken) UpdateLastUsed(ip string) {
	now := utils.NowUnix()
	t.LastUsedAt = &now
	t.LastUsedIP = ip
	t.UpdatedAt = now
}

// Revoke revokes the token
func (t *APIToken) Revoke() {
	t.Status = domaintypes.StatusInactive
	t.UpdatedAt = utils.NowUnix()
}

// BeforeAppendModel implements the bun.BeforeAppendModelHook interface
func (t *APIToken) BeforeAppendModel(_ context.Context, q bun.Query) error {
	now := utils.NowUnix()

	switch q.(type) {
	case *bun.InsertQuery:
		if t.ID.IsNil() {
			t.ID = pulid.MustNew("tok_")
		}
		t.CreatedAt = now
		t.UpdatedAt = now
	case *bun.UpdateQuery:
		t.UpdatedAt = now
	}

	return nil
}

// GetDisplayName returns a display name for the token
func (t *APIToken) GetDisplayName() string {
	if t.Name != "" {
		return fmt.Sprintf("%s (%s...)", t.Name, t.TokenPrefix)
	}
	return fmt.Sprintf("Token %s...", t.TokenPrefix)
}

// SanitizeForResponse removes sensitive data before sending to client
func (t *APIToken) SanitizeForResponse() {
	t.TokenHash = ""
	t.PlainToken = "" // Ensure plain token is never sent after creation
}
