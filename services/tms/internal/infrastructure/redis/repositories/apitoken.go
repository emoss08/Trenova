package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/redis"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	apiTokenPrefix         = "api_token:"
	apiTokenByPrefixPrefix = "api_token_prefix:"
	userAPITokensPrefix    = "user_api_tokens:"  //nolint:gosec // This is a valid prefix
	orgAPITokensPrefix     = "org_api_tokens:"   //nolint:gosec // This is a valid prefix
	defaultAPITokenTTL     = 30 * 24 * time.Hour // 30 days default
)

type APITokenRepositoryParams struct {
	fx.In

	Cache  *redis.Connection
	Logger *zap.Logger
}

type apiTokenRepository struct {
	cache *redis.Connection
	l     *zap.Logger
}

func NewAPITokenRepository(p APITokenRepositoryParams) repositories.APITokenRepository {
	return &apiTokenRepository{
		cache: p.Cache,
		l:     p.Logger.Named("redis.apitoken-repository"),
	}
}

func (r *apiTokenRepository) Create(
	ctx context.Context,
	req repositories.CreateAPITokenRequest,
) error {
	log := r.l.With(zap.String("operation", "Create"))

	token := req.Token

	ttl := defaultAPITokenTTL
	if token.ExpiresAt != nil {
		expiresIn := time.Until(time.Unix(*token.ExpiresAt, 0))
		if expiresIn > 0 {
			ttl = expiresIn
		}
	}

	pipe := r.cache.Client().Pipeline()

	tokenKey := r.getTokenKey(token.ID)
	if err := r.cache.SetJSON(ctx, tokenKey, token, ttl); err != nil {
		log.Error("failed to set token in cache", zap.Error(err))
		return err
	}

	prefixKey := r.getTokenByPrefixKey(token.TokenPrefix)
	pipe.Set(ctx, prefixKey, token.ID.String(), ttl)

	userTokensKey := r.getUserTokensKey(token.UserID)
	pipe.SAdd(ctx, userTokensKey, token.ID.String())
	pipe.Expire(ctx, userTokensKey, ttl)

	orgTokensKey := r.getOrgTokensKey(token.OrganizationID)
	pipe.SAdd(ctx, orgTokensKey, token.ID.String())
	pipe.Expire(ctx, orgTokensKey, ttl)

	if _, err := pipe.Exec(ctx); err != nil {
		log.Error("failed to execute pipeline", zap.Error(err))
		return err
	}

	log.Info("API token created successfully",
		zap.String("tokenId", token.ID.String()),
		zap.String("userId", token.UserID.String()),
		zap.String("orgId", token.OrganizationID.String()),
	)

	return nil
}

func (r *apiTokenRepository) FindByID(
	ctx context.Context,
	tokenID pulid.ID,
) (*tenant.APIToken, error) {
	log := r.l.With(zap.String("operation", "FindByID"))

	token := new(tenant.APIToken)
	if err := r.cache.GetJSON(ctx, r.getTokenKey(tokenID), token); err != nil {
		log.Debug("token not found by ID", zap.String("tokenId", tokenID.String()))
		return nil, errortypes.NewNotFoundError("API token not found")
	}

	return token, nil
}

func (r *apiTokenRepository) FindByToken(
	ctx context.Context,
	req repositories.FindAPITokenByTokenRequest,
) (*tenant.APIToken, error) {
	log := r.l.With(zap.String("operation", "FindByToken"))

	prefixKey := r.getTokenByPrefixKey(req.TokenPrefix)
	tokenIDStr, err := r.cache.Get(ctx, prefixKey)
	if err != nil || tokenIDStr == "" {
		log.Debug("token not found by prefix", zap.String("prefix", req.TokenPrefix))
		return nil, errortypes.NewAuthenticationError("Invalid token")
	}

	tokenID, err := pulid.Parse(tokenIDStr)
	if err != nil {
		log.Error("failed to parse token ID", zap.Error(err))
		return nil, errortypes.NewAuthenticationError("Invalid token")
	}

	token, err := r.FindByID(ctx, tokenID)
	if err != nil {
		return nil, err
	}

	if err = token.VerifyToken(req.PlainToken); err != nil {
		log.Debug("token verification failed", zap.String("tokenId", tokenID.String()))
		return nil, err
	}

	if !token.IsActive() || token.IsExpired() {
		return nil, errortypes.NewAuthenticationError("Token is inactive or expired")
	}

	return token, nil
}

func (r *apiTokenRepository) FindByUserID(
	ctx context.Context,
	userID pulid.ID,
) ([]*tenant.APIToken, error) {
	log := r.l.With(zap.String("operation", "FindByUserID"))

	tokenIDs, err := r.cache.SMembers(ctx, r.getUserTokensKey(userID))
	if err != nil {
		log.Error("failed to get user API tokens", zap.Error(err))
		return nil, err
	}

	tokens := make([]*tenant.APIToken, 0, len(tokenIDs))
	for _, tID := range tokenIDs {
		tokenID, parseErr := pulid.Parse(tID)
		if parseErr != nil {
			continue
		}

		token := new(tenant.APIToken)
		if err = r.cache.GetJSON(ctx, r.getTokenKey(tokenID), token); err != nil {
			_ = r.cache.SRem(ctx, r.getUserTokensKey(userID), tID)
			continue
		}

		if token.IsActive() && !token.IsExpired() {
			tokens = append(tokens, token)
		} else {
			_ = r.cache.SRem(ctx, r.getUserTokensKey(userID), tID)
		}
	}

	return tokens, nil
}

func (r *apiTokenRepository) List(
	ctx context.Context,
	req repositories.ListAPITokensRequest,
) ([]*tenant.APIToken, error) {
	log := r.l.With(zap.String("operation", "List"))

	var tokenIDs []string
	var err error

	switch {
	case req.UserID != nil:
		tokenIDs, err = r.cache.SMembers(ctx, r.getUserTokensKey(*req.UserID))
	case req.OrganizationID != nil:
		tokenIDs, err = r.cache.SMembers(ctx, r.getOrgTokensKey(*req.OrganizationID))
	default:
		return nil, errortypes.NewValidationError(
			"filter",
			errortypes.ErrRequired,
			"Either UserID or OrganizationID must be provided",
		)
	}

	if err != nil {
		log.Error("failed to get token IDs", zap.Error(err))
		return nil, err
	}

	tokens := make([]*tenant.APIToken, 0, len(tokenIDs))
	for _, tID := range tokenIDs {
		tokenID, parseErr := pulid.Parse(tID)
		if parseErr != nil {
			continue
		}

		token := new(tenant.APIToken)
		if err = r.cache.GetJSON(ctx, r.getTokenKey(tokenID), token); err != nil {
			continue
		}

		if !req.IncludeExpired && token.IsExpired() {
			continue
		}
		if !req.IncludeRevoked && !token.IsActive() {
			continue
		}

		if req.BusinessUnitID != nil && token.BusinessUnitID != *req.BusinessUnitID {
			continue
		}

		tokens = append(tokens, token)
	}

	start := req.Offset
	end := req.Offset + req.Limit
	if req.Limit == 0 {
		end = len(tokens)
	}
	if start > len(tokens) {
		start = len(tokens)
	}
	if end > len(tokens) {
		end = len(tokens)
	}

	return tokens[start:end], nil
}

func (r *apiTokenRepository) UpdateLastUsed(
	ctx context.Context,
	req repositories.UpdateAPITokenLastUsedRequest,
) error {
	log := r.l.With(zap.String("operation", "UpdateLastUsed"))

	token, err := r.FindByID(ctx, req.TokenID)
	if err != nil {
		return err
	}

	token.UpdateLastUsed(req.IP)

	ttl := defaultAPITokenTTL
	if token.ExpiresAt != nil {
		expiresIn := time.Until(time.Unix(*token.ExpiresAt, 0))
		if expiresIn > 0 {
			ttl = expiresIn
		}
	}

	if err = r.cache.SetJSON(ctx, r.getTokenKey(token.ID), token, ttl); err != nil {
		log.Error("failed to update token last used", zap.Error(err))
		return err
	}

	return nil
}

func (r *apiTokenRepository) Revoke(ctx context.Context, tokenID pulid.ID) error {
	log := r.l.With(zap.String("operation", "Revoke"))

	token, err := r.FindByID(ctx, tokenID)
	if err != nil {
		return err
	}

	token.Revoke()

	ttl := 24 * time.Hour
	if err = r.cache.SetJSON(ctx, r.getTokenKey(token.ID), token, ttl); err != nil {
		log.Error("failed to update revoked token", zap.Error(err))
		return err
	}

	pipe := r.cache.Client().Pipeline()

	prefixKey := r.getTokenByPrefixKey(token.TokenPrefix)
	pipe.Del(ctx, prefixKey)

	userTokensKey := r.getUserTokensKey(token.UserID)
	pipe.SRem(ctx, userTokensKey, token.ID.String())

	orgTokensKey := r.getOrgTokensKey(token.OrganizationID)
	pipe.SRem(ctx, orgTokensKey, token.ID.String())

	if _, err = pipe.Exec(ctx); err != nil {
		log.Error("failed to remove token from indices", zap.Error(err))
	}

	log.Info("API token revoked successfully",
		zap.String("tokenId", token.ID.String()),
		zap.String("userId", token.UserID.String()),
	)

	return nil
}

func (r *apiTokenRepository) Delete(ctx context.Context, tokenID pulid.ID) error {
	log := r.l.With(zap.String("operation", "Delete"))

	token, err := r.FindByID(ctx, tokenID)
	if err != nil {
		log.Error("failed to remove token from indices", zap.Error(err))
	}

	pipe := r.cache.Client().Pipeline()

	tokenKey := r.getTokenKey(tokenID)
	pipe.Del(ctx, tokenKey)

	prefixKey := r.getTokenByPrefixKey(token.TokenPrefix)
	pipe.Del(ctx, prefixKey)

	userTokensKey := r.getUserTokensKey(token.UserID)
	pipe.SRem(ctx, userTokensKey, tokenID.String())

	orgTokensKey := r.getOrgTokensKey(token.OrganizationID)
	pipe.SRem(ctx, orgTokensKey, tokenID.String())

	if _, err = pipe.Exec(ctx); err != nil {
		log.Error("failed to delete token", zap.Error(err))
		return err
	}

	log.Info("API token deleted successfully", zap.String("tokenId", tokenID.String()))
	return nil
}

func (r *apiTokenRepository) CountActiveByUserID(
	ctx context.Context,
	userID pulid.ID,
) (int64, error) {
	tokens, err := r.FindByUserID(ctx, userID)
	if err != nil {
		return 0, err
	}

	var count int64
	for _, token := range tokens {
		if token.IsActive() && !token.IsExpired() {
			count++
		}
	}

	return count, nil
}

func (r *apiTokenRepository) getTokenKey(tokenID pulid.ID) string {
	return fmt.Sprintf("%s%s", apiTokenPrefix, tokenID.String())
}

func (r *apiTokenRepository) getTokenByPrefixKey(prefix string) string {
	return fmt.Sprintf("%s%s", apiTokenByPrefixPrefix, prefix)
}

func (r *apiTokenRepository) getUserTokensKey(userID pulid.ID) string {
	return fmt.Sprintf("%s%s", userAPITokensPrefix, userID.String())
}

func (r *apiTokenRepository) getOrgTokensKey(orgID pulid.ID) string {
	return fmt.Sprintf("%s%s", orgAPITokensPrefix, orgID.String())
}
