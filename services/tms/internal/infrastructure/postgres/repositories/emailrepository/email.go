package emailrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

const profileReturningColumns = `
	id,
	business_unit_id,
	organization_id,
	name,
	description,
	from_name,
	from_address,
	reply_to,
	provider_type,
	auth_type,
	encryption_type,
	status,
	version,
	created_at,
	updated_at
`

func New(p Params) repositories.EmailRepository {
	return &repository{db: p.DB, l: p.Logger.Named("postgres.email-repository")}
}

func applyTenant(q *bun.SelectQuery, alias string, ti pagination.TenantInfo) *bun.SelectQuery {
	return q.Where(alias+".organization_id = ?", ti.OrgID).
		Where(alias+".business_unit_id = ?", ti.BuID)
}

func (r *repository) ListProfiles(
	ctx context.Context,
	req *repositories.ListEmailProfilesRequest,
) (*pagination.ListResult[*email.Profile], error) {
	entities := make([]*email.Profile, 0, req.Filter.Pagination.SafeLimit())
	total, err := r.db.DB().NewSelect().
		Model(&entities).
		Apply(func(q *bun.SelectQuery) *bun.SelectQuery {
			q = applyTenant(q, "ep", req.Filter.TenantInfo)
			q = querybuilder.ApplyFilters(q, "ep", req.Filter, (*email.Profile)(nil))
			return q.Limit(req.Filter.Pagination.SafeLimit()).
				Offset(req.Filter.Pagination.SafeOffset())
		}).
		OrderExpr("ep.created_at DESC").
		ScanAndCount(ctx)
	if err != nil {
		return nil, err
	}
	return &pagination.ListResult[*email.Profile]{Items: entities, Total: total}, nil
}

func (r *repository) SelectProfileOptions(
	ctx context.Context,
	req *repositories.EmailProfileSelectOptionsRequest,
) (*pagination.ListResult[*email.Profile], error) {
	return dbhelper.SelectOptions[*email.Profile](
		ctx,
		r.db.DB(),
		req.SelectQueryRequest,
		&dbhelper.SelectOptionsConfig{
			Columns: []string{
				"id",
				"business_unit_id",
				"organization_id",
				"name",
				"from_name",
				"from_address",
				"reply_to",
				"provider_type",
				"status",
				"updated_at",
			},
			OrgColumn: "ep.organization_id",
			BuColumn:  "ep.business_unit_id",
			QueryModifier: func(q *bun.SelectQuery) *bun.SelectQuery {
				return q.Where("ep.status = ?", email.ProfileStatusActive).
					OrderExpr("ep.name ASC")
			},
			EntityName: "EmailProfile",
			SearchColumns: []string{
				"ep.name",
				"ep.from_address",
				"ep.from_name",
			},
		},
	)
}

func (r *repository) GetProfile(
	ctx context.Context,
	req repositories.GetEmailEntityRequest,
) (*email.Profile, error) {
	entity := new(email.Profile)
	err := r.db.DB().NewSelect().
		Model(entity).
		Apply(func(q *bun.SelectQuery) *bun.SelectQuery {
			return applyTenant(q, "ep", req.TenantInfo).Where("ep.id = ?", req.ID)
		}).
		Scan(ctx)
	return entity, dberror.HandleNotFoundError(err, "EmailProfile")
}

func (r *repository) CreateProfile(ctx context.Context, entity *email.Profile) (*email.Profile, error) {
	_, err := r.db.DB().NewInsert().Model(entity).Returning(profileReturningColumns).Exec(ctx)
	return entity, err
}

func (r *repository) UpdateProfile(ctx context.Context, entity *email.Profile) (*email.Profile, error) {
	previousVersion := entity.Version
	entity.Version++
	result, err := r.db.DB().NewUpdate().
		Model(entity).
		WherePK().
		Where("version = ?", previousVersion).
		OmitZero().
		Returning(profileReturningColumns).
		Exec(ctx)
	if err != nil {
		return nil, err
	}
	if err = dberror.CheckRowsAffected(result, "EmailProfile", entity.ID.String()); err != nil {
		return nil, err
	}
	return entity, nil
}

func (r *repository) DeleteProfile(ctx context.Context, req repositories.GetEmailEntityRequest) error {
	result, err := r.db.DB().NewDelete().
		Model((*email.Profile)(nil)).
		Where("id = ?", req.ID).
		Where("organization_id = ?", req.TenantInfo.OrgID).
		Where("business_unit_id = ?", req.TenantInfo.BuID).
		Exec(ctx)
	if err != nil {
		return err
	}
	return dberror.CheckRowsAffected(result, "EmailProfile", req.ID.String())
}

func (r *repository) ListAssignments(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*email.ProfileAssignment, error) {
	entities := make([]*email.ProfileAssignment, 0, len(email.Purposes()))
	err := r.db.DB().NewSelect().
		Model(&entities).
		Relation("Profile").
		Apply(func(q *bun.SelectQuery) *bun.SelectQuery {
			return applyTenant(q, "epa", tenantInfo)
		}).
		OrderExpr("epa.purpose ASC").
		Scan(ctx)
	return entities, err
}

func (r *repository) UpsertAssignments(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	assignments []*email.ProfileAssignment,
) ([]*email.ProfileAssignment, error) {
	err := r.db.DB().RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		purposes := email.Purposes()
		provided := make(map[email.Purpose]struct{}, len(assignments))

		for _, assignment := range assignments {
			if assignment == nil {
				continue
			}
			provided[assignment.Purpose] = struct{}{}
			assignment.OrganizationID = tenantInfo.OrgID
			assignment.BusinessUnitID = tenantInfo.BuID
			_, err := tx.NewInsert().
				Model(assignment).
				On("CONFLICT (organization_id, business_unit_id, purpose) DO UPDATE").
				Set("profile_id = EXCLUDED.profile_id").
				Set("updated_at = EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint").
				Returning("*").
				Exec(ctx)
			if err != nil {
				return err
			}
		}

		missing := make([]email.Purpose, 0, len(purposes))
		for _, purpose := range purposes {
			if _, ok := provided[purpose]; !ok {
				missing = append(missing, purpose)
			}
		}
		if len(missing) == 0 {
			return nil
		}

		_, err := tx.NewDelete().
			Model((*email.ProfileAssignment)(nil)).
			Where("organization_id = ?", tenantInfo.OrgID).
			Where("business_unit_id = ?", tenantInfo.BuID).
			Where("purpose IN (?)", bun.In(missing)).
			Exec(ctx)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	return r.ListAssignments(ctx, tenantInfo)
}

func (r *repository) GetAssignedProfile(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	purpose email.Purpose,
) (*email.Profile, error) {
	entity := new(email.Profile)
	err := r.db.DB().NewSelect().
		Model(entity).
		Join("JOIN email_profile_assignments AS epa ON epa.profile_id = ep.id").
		Where("epa.organization_id = ?", tenantInfo.OrgID).
		Where("epa.business_unit_id = ?", tenantInfo.BuID).
		Where("epa.purpose = ?", purpose).
		Where("ep.status = ?", email.ProfileStatusActive).
		Scan(ctx)
	return entity, dberror.HandleNotFoundError(err, "EmailProfile")
}

func (r *repository) CreateMessage(ctx context.Context, entity *email.Message) (*email.Message, error) {
	_, err := r.db.DB().NewInsert().Model(entity).Returning("*").Exec(ctx)
	return entity, err
}

func (r *repository) UpdateMessage(ctx context.Context, entity *email.Message) (*email.Message, error) {
	_, err := r.db.DB().
		NewUpdate().
		Model(entity).
		WherePK().
		Set("provider_message_id = ?", entity.ProviderMessageID).
		Set("status = ?", entity.Status).
		Set("attempts = ?", entity.Attempts).
		Set("last_error = ?", entity.LastError).
		Set("sent_at = ?", entity.SentAt).
		Set("delivered_at = ?", entity.DeliveredAt).
		Set("failed_at = ?", entity.FailedAt).
		Set("updated_at = ?", entity.UpdatedAt).
		Returning("*").
		Exec(ctx)
	return entity, err
}

func (r *repository) GetMessage(
	ctx context.Context,
	req repositories.GetEmailEntityRequest,
) (*email.Message, error) {
	entity := new(email.Message)
	err := r.db.DB().NewSelect().
		Model(entity).
		Relation("Profile").
		Apply(func(q *bun.SelectQuery) *bun.SelectQuery {
			return applyTenant(q, "em", req.TenantInfo).Where("em.id = ?", req.ID)
		}).
		Scan(ctx)
	return entity, dberror.HandleNotFoundError(err, "EmailMessage")
}

func (r *repository) GetMessageByProviderID(
	ctx context.Context,
	req repositories.GetEmailMessageByProviderIDRequest,
) (*email.Message, error) {
	entity := new(email.Message)
	err := r.db.DB().NewSelect().
		Model(entity).
		Where("em.organization_id = ?", req.TenantInfo.OrgID).
		Where("em.business_unit_id = ?", req.TenantInfo.BuID).
		Where("em.provider = ?", req.Provider).
		Where("em.provider_message_id = ?", req.ProviderMessageID).
		Scan(ctx)
	return entity, dberror.HandleNotFoundError(err, "EmailMessage")
}

func (r *repository) CreateAttachments(
	ctx context.Context,
	entities []*email.Attachment,
) ([]*email.Attachment, error) {
	if len(entities) == 0 {
		return []*email.Attachment{}, nil
	}
	_, err := r.db.DB().NewInsert().Model(&entities).Returning("*").Exec(ctx)
	return entities, err
}

func (r *repository) ListAttachments(
	ctx context.Context,
	req repositories.ListEmailAttachmentsRequest,
) ([]*email.Attachment, error) {
	entities := make([]*email.Attachment, 0)
	err := r.db.DB().NewSelect().
		Model(&entities).
		Where("ema.organization_id = ?", req.TenantInfo.OrgID).
		Where("ema.business_unit_id = ?", req.TenantInfo.BuID).
		Where("ema.message_id = ?", req.MessageID).
		OrderExpr("ema.created_at ASC").
		Scan(ctx)
	return entities, err
}

func (r *repository) GetEmailWebhookConfig(
	ctx context.Context,
	req repositories.GetEmailWebhookConfigRequest,
) (*repositories.EmailWebhookConfig, error) {
	var row struct {
		OrganizationID       string `bun:"organization_id"`
		BusinessUnitID       string `bun:"business_unit_id"`
		WebhookSigningSecret string `bun:"webhook_signing_secret"`
	}
	err := r.db.DB().NewSelect().
		TableExpr("integrations AS integ").
		ColumnExpr("integ.organization_id").
		ColumnExpr("integ.business_unit_id").
		ColumnExpr("integ.configuration ->> 'webhookSigningSecret' AS webhook_signing_secret").
		Where("integ.type = ?", req.IntegrationType).
		Where("integ.configuration ->> 'webhookToken' = ?", req.Token).
		Where("integ.enabled = ?", true).
		Scan(ctx, &row)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "EmailWebhook")
	}

	orgID, err := pulid.Parse(row.OrganizationID)
	if err != nil {
		return nil, fmt.Errorf("parse webhook organization id: %w", err)
	}
	buID, err := pulid.Parse(row.BusinessUnitID)
	if err != nil {
		return nil, fmt.Errorf("parse webhook business unit id: %w", err)
	}

	return &repositories.EmailWebhookConfig{
		TenantInfo: pagination.TenantInfo{
			OrgID: orgID,
			BuID:  buID,
		},
		SigningSecret: row.WebhookSigningSecret,
	}, nil
}

func (r *repository) ListMessages(
	ctx context.Context,
	req *repositories.ListEmailMessagesRequest,
) (*pagination.ListResult[*email.Message], error) {
	entities := make([]*email.Message, 0, req.Filter.Pagination.SafeLimit())
	total, err := r.db.DB().NewSelect().
		Model(&entities).
		Relation("Profile").
		Apply(func(q *bun.SelectQuery) *bun.SelectQuery {
			q = applyTenant(q, "em", req.Filter.TenantInfo)
			q = querybuilder.ApplyFilters(q, "em", req.Filter, (*email.Message)(nil))
			return q.Limit(req.Filter.Pagination.SafeLimit()).
				Offset(req.Filter.Pagination.SafeOffset())
		}).
		OrderExpr("em.created_at DESC").
		ScanAndCount(ctx)
	if err != nil {
		return nil, err
	}
	return &pagination.ListResult[*email.Message]{Items: entities, Total: total}, nil
}

func (r *repository) CreateEvent(ctx context.Context, entity *email.Event) (bool, error) {
	result, err := r.db.DB().NewInsert().
		Model(entity).
		On("CONFLICT (organization_id, business_unit_id, provider, provider_event_id) DO NOTHING").
		Exec(ctx)
	if err != nil {
		return false, err
	}
	rows, err := result.RowsAffected()
	return rows > 0, err
}

func (r *repository) ListSuppressions(
	ctx context.Context,
	req *repositories.ListEmailSuppressionsRequest,
) (*pagination.ListResult[*email.Suppression], error) {
	entities := make([]*email.Suppression, 0, req.Filter.Pagination.SafeLimit())
	total, err := r.db.DB().NewSelect().
		Model(&entities).
		Apply(func(q *bun.SelectQuery) *bun.SelectQuery {
			return applyTenant(q, "es", req.Filter.TenantInfo).
				Limit(req.Filter.Pagination.SafeLimit()).
				Offset(req.Filter.Pagination.SafeOffset())
		}).
		OrderExpr("es.created_at DESC").
		ScanAndCount(ctx)
	if err != nil {
		return nil, err
	}
	return &pagination.ListResult[*email.Suppression]{Items: entities, Total: total}, nil
}

func (r *repository) CreateSuppression(
	ctx context.Context,
	entity *email.Suppression,
) (*email.Suppression, error) {
	entity.EmailAddress = stringutils.NormalizeEmailAddress(entity.EmailAddress)
	_, err := r.db.DB().NewInsert().
		Model(entity).
		On("CONFLICT (organization_id, business_unit_id, email_address) DO UPDATE").
		Set("reason = EXCLUDED.reason").
		Set("provider = EXCLUDED.provider").
		Set("source_event_id = EXCLUDED.source_event_id").
		Set("notes = EXCLUDED.notes").
		Returning("*").
		Exec(ctx)
	return entity, err
}

func (r *repository) DeleteSuppression(ctx context.Context, req repositories.GetEmailEntityRequest) error {
	result, err := r.db.DB().NewDelete().
		Model((*email.Suppression)(nil)).
		Where("id = ?", req.ID).
		Where("organization_id = ?", req.TenantInfo.OrgID).
		Where("business_unit_id = ?", req.TenantInfo.BuID).
		Exec(ctx)
	if err != nil {
		return err
	}
	return dberror.CheckRowsAffected(result, "EmailSuppression", req.ID.String())
}

func (r *repository) HasSuppression(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	emailAddress string,
) (bool, error) {
	count, err := r.db.DB().NewSelect().
		Model((*email.Suppression)(nil)).
		Where("organization_id = ?", tenantInfo.OrgID).
		Where("business_unit_id = ?", tenantInfo.BuID).
		Where("email_address = ?", stringutils.NormalizeEmailAddress(emailAddress)).
		Count(ctx)
	return count > 0, err
}
