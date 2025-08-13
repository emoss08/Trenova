package repositories

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/shared/edi/internal/core/domain"
	"github.com/emoss08/trenova/shared/edi/internal/core/ports"
	"github.com/emoss08/trenova/shared/edi/internal/infrastructure/database"
	"go.uber.org/fx"
)

type ediPartnerProfileRepository struct {
	db *database.DB
}

type EDIPartnerProfileRepoParams struct {
	fx.In
	DB *database.DB
}

func NewEDIPartnerProfileRepository(params EDIPartnerProfileRepoParams) ports.EDIPartnerProfileRepository {
	return &ediPartnerProfileRepository{
		db: params.DB,
	}
}

func (r *ediPartnerProfileRepository) Create(ctx context.Context, profile *domain.EDIPartnerProfile) error {
	_, err := r.db.NewInsert().Model(profile).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create EDI partner profile: %w", err)
	}
	return nil
}

func (r *ediPartnerProfileRepository) GetByPartnerID(ctx context.Context, partnerID string) (*domain.EDIPartnerProfile, error) {
	profile := new(domain.EDIPartnerProfile)
	err := r.db.NewSelect().
		Model(profile).
		Where("partner_id = ?", partnerID).
		Scan(ctx)
	
	if err != nil {
		return nil, fmt.Errorf("failed to get EDI partner profile by partner ID: %w", err)
	}
	return profile, nil
}

func (r *ediPartnerProfileRepository) Update(ctx context.Context, profile *domain.EDIPartnerProfile) error {
	_, err := r.db.NewUpdate().
		Model(profile).
		WherePK().
		Exec(ctx)
	
	if err != nil {
		return fmt.Errorf("failed to update EDI partner profile: %w", err)
	}
	return nil
}

func (r *ediPartnerProfileRepository) List(ctx context.Context, active bool) ([]*domain.EDIPartnerProfile, error) {
	var profiles []*domain.EDIPartnerProfile
	
	query := r.db.NewSelect().Model(&profiles).OrderExpr("partner_name ASC")
	
	if active {
		query = query.Where("active = ?", true)
	}
	
	err := query.Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list EDI partner profiles: %w", err)
	}
	
	return profiles, nil
}

func (r *ediPartnerProfileRepository) Delete(ctx context.Context, partnerID string) error {
	_, err := r.db.NewDelete().
		Model((*domain.EDIPartnerProfile)(nil)).
		Where("partner_id = ?", partnerID).
		Exec(ctx)
	
	if err != nil {
		return fmt.Errorf("failed to delete EDI partner profile: %w", err)
	}
	return nil
}