package emailrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email/emaildb"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration/integrationdb"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var (
	profileErrors     = dberror.Mapping{Entity: "EmailProfile"}
	messageErrors     = dberror.Mapping{Entity: "EmailMessage"}
	webhookErrors     = dberror.Mapping{Entity: "EmailWebhook"}
	suppressionErrors = dberror.Mapping{Entity: "EmailSuppression"}
)

type webhookRow struct {
	OrganizationID       pulid.ID
	BusinessUnitID       pulid.ID
	WebhookSigningSecret string
}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) emailports.EmailRepository {
	return &repository{db: p.DB, l: p.Logger.Named("postgres.email-repository")}
}

func (r *repository) ListProfiles(
	ctx context.Context,
	req *emailports.ListEmailProfilesRequest,
) (*pagination.ListResult[*email.Profile], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*email.Profile], error) {
			p := emaildb.Profiles
			return listquery.List(
				ctx,
				r.db.Keel(ctx),
				p.Query(),
				&listquery.Request[email.Profile]{
					Filter: req.Filter,
					Then:   []keel.Order{p.CreatedAt.Desc()},
				},
			)
		},
	)
}

func (r *repository) ListProfilesConnection(
	ctx context.Context,
	req *emailports.ListEmailProfileConnectionRequest,
) (*pagination.CursorListResult[*email.Profile], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*email.Profile], error) {
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				emaildb.Profiles.Query(),
				&listquery.Request[email.Profile]{
					Filter:  req.Filter,
					Cursor:  req.Cursor,
					Columns: req.EmailProfileColumns,
				},
			)
			if err != nil {
				r.l.Error("failed to list email profile connection", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) SelectProfileOptions(
	ctx context.Context,
	req *emailports.EmailProfileSelectOptionsRequest,
) (*pagination.ListResult[*email.Profile], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*email.Profile], error) {
			p := emaildb.Profiles
			q := p.Pick(
				p.ID,
				p.BusinessUnitID,
				p.OrganizationID,
				p.Name,
				p.SenderName,
				p.SenderEmail,
				p.ReplyToEmail,
				p.Provider,
				p.Status,
				p.UpdatedAt,
			).
				Where(p.Status.Eq(email.ProfileStatusActive)).
				Order(p.Name.Asc())

			return listquery.SelectOptions(
				ctx,
				r.db.Keel(ctx),
				q,
				req.SelectQueryRequest,
				func(s string) keel.Pred {
					return keel.Or(
						p.Name.ContainsFold(s),
						p.SenderEmail.ContainsFold(s),
						p.SenderName.ContainsFold(s),
					)
				},
			)
		},
	)
}

func (r *repository) GetProfile(
	ctx context.Context,
	req emailports.GetEmailEntityRequest,
) (*email.Profile, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*email.Profile, error) {
		entity, err := emaildb.Profiles.Get(ctx, r.db.Keel(ctx), req.ID)
		if err != nil {
			return nil, profileErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) CreateProfile(
	ctx context.Context,
	entity *email.Profile,
) (*email.Profile, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*email.Profile, error) {
		return entity, emaildb.Profiles.Create(ctx, r.db.Keel(ctx), entity)
	})
}

func (r *repository) UpdateProfile(
	ctx context.Context,
	entity *email.Profile,
) (*email.Profile, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*email.Profile, error) {
		p := emaildb.Profiles
		db := r.db.Keel(ctx)

		var err error
		switch {
		case entity.AuthType == "" && entity.EncryptionType == "":
			err = p.SaveExcept(ctx, db, entity, p.AuthType, p.EncryptionType)
		case entity.AuthType == "":
			err = p.SaveExcept(ctx, db, entity, p.AuthType)
		case entity.EncryptionType == "":
			err = p.SaveExcept(ctx, db, entity, p.EncryptionType)
		default:
			err = p.Save(ctx, db, entity)
		}
		if err != nil {
			return nil, profileErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) DeleteProfile(
	ctx context.Context,
	req emailports.GetEmailEntityRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		p := emaildb.Profiles
		rows, err := p.Where(p.ID.Eq(req.ID)).Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			return err
		}

		return dberror.CheckAffected(rows, "EmailProfile", req.ID.String())
	})
}

func (r *repository) ListAssignments(
	ctx context.Context,
	_ pagination.TenantInfo,
) ([]*email.ProfileAssignment, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*email.ProfileAssignment, error) {
		a := emaildb.ProfileAssignments
		return a.Load(a.Profile()).Order(a.Purpose.Asc()).All(ctx, r.db.Keel(ctx))
	})
}

func (r *repository) UpsertAssignments(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	assignments []*email.ProfileAssignment,
) ([]*email.ProfileAssignment, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) ([]*email.ProfileAssignment, error) {
		err := dbtx.Savepoint(ctx, r.db, func(ctx context.Context) error {
			a := emaildb.ProfileAssignments
			db := r.db.Keel(ctx)

			rows := make([]*email.ProfileAssignment, 0, len(assignments))
			provided := make(map[email.Purpose]struct{}, len(assignments))
			for _, assignment := range assignments {
				if assignment == nil {
					continue
				}
				provided[assignment.Purpose] = struct{}{}
				assignment.OrganizationID = tenantInfo.OrgID
				assignment.BusinessUnitID = tenantInfo.BuID
				rows = append(rows, assignment)
			}

			if len(rows) > 0 {
				if _, err := a.Upsert(
					ctx,
					db,
					rows,
					keel.OnConflict(a.OrganizationID, a.BusinessUnitID, a.Purpose).
						Update(a.ProfileID),
				); err != nil {
					return err
				}
			}

			missing := make([]email.Purpose, 0, len(email.Purposes()))
			for _, purpose := range email.Purposes() {
				if _, ok := provided[purpose]; !ok {
					missing = append(missing, purpose)
				}
			}
			if len(missing) == 0 {
				return nil
			}

			_, err := a.Where(a.Purpose.In(missing...)).Delete(ctx, db)
			return err
		})
		if err != nil {
			return nil, err
		}

		return r.ListAssignments(ctx, tenantInfo)
	})
}

func (r *repository) GetAssignedProfile(
	ctx context.Context,
	_ pagination.TenantInfo,
	purpose email.Purpose,
) (*email.Profile, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*email.Profile, error) {
		p := emaildb.Profiles
		a := emaildb.ProfileAssignments
		entity, err := p.Join(a, a.ProfileID.EqCol(p.ID)).
			Where(a.Purpose.Eq(purpose), p.Status.Eq(email.ProfileStatusActive)).
			Limit(1).
			First(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, profileErrors.Map(err, string(purpose))
		}

		return entity, nil
	})
}

func (r *repository) CreateMessage(
	ctx context.Context,
	entity *email.Message,
) (*email.Message, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*email.Message, error) {
		return entity, emaildb.Messages.Create(ctx, r.db.Keel(ctx), entity)
	})
}

func (r *repository) CreateMessageOnce(
	ctx context.Context,
	entity *email.Message,
) (*email.Message, bool, error) {
	return dbtx.Write2(ctx, r.db, func(ctx context.Context) (*email.Message, bool, error) {
		m := emaildb.Messages
		db := r.db.Keel(ctx)
		inserted, err := m.CreateOrSkip(
			ctx,
			db,
			[]*email.Message{entity},
			m.OrganizationID,
			m.BusinessUnitID,
			m.IdempotencyKey,
		)
		if err != nil {
			return nil, false, err
		}
		if inserted > 0 {
			return entity, true, nil
		}

		existing, err := m.Where(m.IdempotencyKey.Eq(entity.IdempotencyKey)).One(ctx, db)
		if err != nil {
			return nil, false, messageErrors.Map(err, entity.IdempotencyKey)
		}

		return existing, false, nil
	})
}

func (r *repository) UpdateMessage(
	ctx context.Context,
	entity *email.Message,
) (*email.Message, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*email.Message, error) {
		m := emaildb.Messages
		err := m.Save(
			ctx,
			r.db.Keel(ctx),
			entity,
			m.ProviderMessageID,
			m.Status,
			m.Attempts,
			m.LastError,
			m.SentAt,
			m.DeliveredAt,
			m.FailedAt,
		)
		if err != nil {
			return nil, messageErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) GetMessage(
	ctx context.Context,
	req emailports.GetEmailEntityRequest,
) (*email.Message, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*email.Message, error) {
		m := emaildb.Messages
		entity, err := m.Where(m.ID.Eq(req.ID)).Load(m.Profile()).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, messageErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) GetMessageByProviderID(
	ctx context.Context,
	req emailports.GetEmailMessageByProviderIDRequest,
) (*email.Message, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*email.Message, error) {
		m := emaildb.Messages
		entity, err := m.Where(
			m.Provider.Eq(req.Provider),
			m.ProviderMessageID.Eq(req.ProviderMessageID),
		).First(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, messageErrors.Map(err, req.ProviderMessageID)
		}

		return entity, nil
	})
}

func (r *repository) CreateAttachments(
	ctx context.Context,
	entities []*email.Attachment,
) ([]*email.Attachment, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) ([]*email.Attachment, error) {
		if len(entities) == 0 {
			return []*email.Attachment{}, nil
		}

		return entities, emaildb.Attachments.CreateMany(ctx, r.db.Keel(ctx), entities)
	})
}

func (r *repository) ListAttachments(
	ctx context.Context,
	req emailports.ListEmailAttachmentsRequest,
) ([]*email.Attachment, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*email.Attachment, error) {
		a := emaildb.Attachments
		return a.Where(a.MessageID.Eq(req.MessageID)).
			Order(a.CreatedAt.Asc()).
			All(ctx, r.db.Keel(ctx))
	})
}

func (r *repository) GetEmailWebhookConfig(
	ctx context.Context,
	req emailports.GetEmailWebhookConfigRequest,
) (*emailports.EmailWebhookConfig, error) {
	ctx = dbscope.WithSystem(
		ctx,
		"resolve the email provider webhook that was called before its tenant is known",
	)
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*emailports.EmailWebhookConfig, error) {
		i := integrationdb.Integrations
		row, err := keel.IntoOne[webhookRow](
			ctx,
			r.db.Keel(ctx),
			i.AllTenants().
				Pick(
					i.OrganizationID,
					i.BusinessUnitID,
					keel.DynOf(i.Configuration.Text("webhookSigningSecret")).Coalesce(""),
				).
				Where(
					i.Type.Eq(req.IntegrationType),
					i.Configuration.Text("webhookToken").Eq(req.Token),
					i.Enabled.IsTrue(),
				).
				Limit(1),
		)
		if err != nil {
			return nil, webhookErrors.Map(err, string(req.IntegrationType))
		}

		return &emailports.EmailWebhookConfig{
			TenantInfo: pagination.TenantInfo{
				OrgID: row.OrganizationID,
				BuID:  row.BusinessUnitID,
			},
			SigningSecret: row.WebhookSigningSecret,
		}, nil
	})
}

func (r *repository) ListMessages(
	ctx context.Context,
	req *emailports.ListEmailMessagesRequest,
) (*pagination.ListResult[*email.Message], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*email.Message], error) {
			m := emaildb.Messages
			return listquery.List(
				ctx,
				r.db.Keel(ctx),
				m.Query(),
				&listquery.Request[email.Message]{
					Filter: req.Filter,
					Shape: func(q *keel.Query[*email.Message]) *keel.Query[*email.Message] {
						return q.Load(m.Profile())
					},
					Then: []keel.Order{m.CreatedAt.Desc()},
				},
			)
		},
	)
}

func (r *repository) CreateEvent(ctx context.Context, entity *email.Event) (bool, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (bool, error) {
		e := emaildb.Events
		inserted, err := e.CreateOrSkip(
			ctx,
			r.db.Keel(ctx),
			[]*email.Event{entity},
			e.OrganizationID,
			e.BusinessUnitID,
			e.Provider,
			e.ProviderEventID,
		)

		return inserted > 0, err
	})
}

//nolint:gocritic // Repository request structs follow the existing value-parameter port contracts.
func (r *repository) ListEventRecipients(
	ctx context.Context,
	req emailports.ListEmailEventRecipientsRequest,
) ([]string, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]string, error) {
		e := emaildb.Events
		recipient := keel.DynFunc("lower", e.Recipient)
		recipients, err := keel.Into[string](
			ctx,
			r.db.Keel(ctx),
			e.Where(
				e.MessageID.Eq(req.MessageID),
				e.Type.Eq(req.Type),
				e.Recipient.IsNotNull(),
			).Group(recipient).Pick(recipient),
		)
		if err != nil {
			return nil, err
		}
		if recipients == nil {
			recipients = []string{}
		}

		return recipients, nil
	})
}

func (r *repository) ListSuppressions(
	ctx context.Context,
	req *emailports.ListEmailSuppressionsRequest,
) (*pagination.ListResult[*email.Suppression], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*email.Suppression], error) {
			s := emaildb.Suppressions
			page := req.Filter.Pagination
			rows := s.Order(s.CreatedAt.Desc()).Limit(page.SafeLimit()).Offset(page.SafeOffset())

			var items []*email.Suppression
			var total int64
			if err := keel.Batch(
				ctx,
				r.db.Keel(ctx),
				keel.ListInto(rows, &items),
				keel.CountInto(s.Query(), &total),
			); err != nil {
				return nil, err
			}

			return &pagination.ListResult[*email.Suppression]{Items: items, Total: int(total)}, nil
		},
	)
}

func (r *repository) CreateSuppression(
	ctx context.Context,
	entity *email.Suppression,
) (*email.Suppression, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*email.Suppression, error) {
		s := emaildb.Suppressions
		db := r.db.Keel(ctx)
		entity.EmailAddress = stringutils.NormalizeEmailAddress(entity.EmailAddress)
		inserted, err := s.Upsert(
			ctx,
			db,
			[]*email.Suppression{entity},
			keel.OnConflict(s.OrganizationID, s.BusinessUnitID, s.EmailAddress).
				Update(s.Reason, s.Provider, s.SourceEventID, s.Notes),
		)
		if err != nil {
			return nil, err
		}
		if inserted[0] {
			return entity, nil
		}

		stored, err := s.Where(s.EmailAddress.Eq(entity.EmailAddress)).One(ctx, db)
		if err != nil {
			return nil, suppressionErrors.Map(err, entity.EmailAddress)
		}

		return stored, nil
	})
}

func (r *repository) DeleteSuppression(
	ctx context.Context,
	req emailports.GetEmailEntityRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		s := emaildb.Suppressions
		rows, err := s.Where(s.ID.Eq(req.ID)).Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			return err
		}

		return dberror.CheckAffected(rows, "EmailSuppression", req.ID.String())
	})
}

func (r *repository) HasSuppression(
	ctx context.Context,
	_ pagination.TenantInfo,
	emailAddress string,
) (bool, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (bool, error) {
		s := emaildb.Suppressions
		return s.Where(s.EmailAddress.Eq(stringutils.NormalizeEmailAddress(emailAddress))).
			Exists(ctx, r.db.Keel(ctx))
	})
}
