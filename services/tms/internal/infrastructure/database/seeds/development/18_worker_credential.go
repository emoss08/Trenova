package development

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

const credentialSeedDay = int64(86400)

type WorkerCredentialSeed struct {
	seedhelpers.BaseSeed
}

// WorkerCredentialSeed gives every seeded driver a realistic credential file:
// the system catalog, a row per mirrored profile column, and a spread of
// healthy, expiring and expired optional credentials so the Credentials tab,
// the expiry forecast and the nightly sweep all have something to show.
//
// Depends on:
//   - Worker: the drivers whose profiles are mirrored
func NewWorkerCredentialSeed() *WorkerCredentialSeed {
	seed := &WorkerCredentialSeed{}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"WorkerCredential",
		"1.0.0",
		"Seeds credential types and per-worker credentials mirrored from worker profiles",
		[]common.Environment{common.EnvDevelopment},
	)
	seed.SetDependencies(seedhelpers.SeedWorker)
	return seed
}

type credentialSeedRefs struct {
	orgID   pulid.ID
	buID    pulid.ID
	adminID pulid.ID
	now     int64
	types   map[string]*worker.WorkerCredentialType
}

func (s *WorkerCredentialSeed) Run(ctx context.Context, tx bun.Tx) error {
	return seedhelpers.RunInTransaction(
		ctx,
		tx,
		s.Name(),
		nil,
		func(ctx context.Context, tx bun.Tx, sc *seedhelpers.SeedContext) error {
			org, err := sc.GetDefaultOrganization(ctx)
			if err != nil {
				return err
			}
			admin, err := sc.GetUserByUsername(ctx, "admin")
			if err != nil {
				return fmt.Errorf("get admin user: %w", err)
			}
			refs := &credentialSeedRefs{
				orgID:   org.ID,
				buID:    org.BusinessUnitID,
				adminID: admin.ID,
				now:     timeutils.NowUnix(),
			}

			if err = s.ensureTypes(ctx, tx, refs); err != nil {
				return fmt.Errorf("ensure credential types: %w", err)
			}

			workers, err := s.loadWorkers(ctx, tx, refs)
			if err != nil {
				return fmt.Errorf("load workers: %w", err)
			}
			for i, wrk := range workers {
				if err = s.seedWorker(ctx, tx, sc, refs, wrk, i); err != nil {
					return fmt.Errorf("seed credentials for worker %s: %w", wrk.ID, err)
				}
			}

			return nil
		},
	)
}

func (s *WorkerCredentialSeed) ensureTypes(
	ctx context.Context,
	tx bun.Tx,
	refs *credentialSeedRefs,
) error {
	rows := worker.SystemCredentialTypes()
	for _, row := range rows {
		row.OrganizationID = refs.orgID
		row.BusinessUnitID = refs.buID
		row.Status = domaintypes.StatusActive
		row.IsSystem = true
		row.RequiredForDriverTypes = []worker.DriverType{}
	}
	if _, err := tx.NewInsert().Model(&rows).On("CONFLICT DO NOTHING").Exec(ctx); err != nil {
		return err
	}

	existing := make([]*worker.WorkerCredentialType, 0, len(rows))
	cols := buncolgen.WorkerCredentialTypeColumns
	if err := tx.NewSelect().
		Model(&existing).
		Where(cols.OrganizationID.Eq(), refs.orgID).
		Where(cols.BusinessUnitID.Eq(), refs.buID).
		Scan(ctx); err != nil {
		return err
	}
	refs.types = make(map[string]*worker.WorkerCredentialType, len(existing))
	for _, typ := range existing {
		refs.types[typ.Code] = typ
	}
	return nil
}

func (s *WorkerCredentialSeed) loadWorkers(
	ctx context.Context,
	tx bun.Tx,
	refs *credentialSeedRefs,
) ([]*worker.Worker, error) {
	workers := make([]*worker.Worker, 0, 32)
	cols := buncolgen.WorkerColumns
	err := tx.NewSelect().
		Model(&workers).
		Relation(buncolgen.WorkerRelations.Profile, func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Relation(buncolgen.WorkerProfileRelations.LicenseState)
		}).
		Where(cols.OrganizationID.Eq(), refs.orgID).
		Where(cols.BusinessUnitID.Eq(), refs.buID).
		Order(cols.CreatedAt.OrderAsc()).
		Scan(ctx)
	return workers, err
}

type seedCredential struct {
	code       string
	expiresIn  int64
	issuedAgo  int64
	number     string
	authority  string
	verified   bool
	noExpiry   bool
	notes      string
	profileSet bool
}

func (s *WorkerCredentialSeed) plan(wrk *worker.Worker, index int) []seedCredential {
	profile := wrk.Profile
	plan := make([]seedCredential, 0, 8)
	if profile == nil {
		return plan
	}

	authority := ""
	if profile.LicenseState != nil {
		authority = profile.LicenseState.Abbreviation
	}
	plan = append(plan, seedCredential{
		code:       "CDL",
		number:     profile.LicenseNumber,
		authority:  authority,
		verified:   true,
		profileSet: true,
	})

	medicalDays := int64(400)
	switch index % 5 {
	case 1:
		medicalDays = 20
	case 3:
		medicalDays = -4
	}
	plan = append(plan, seedCredential{
		code:      "MED_CARD",
		expiresIn: medicalDays,
		issuedAgo: 730 - medicalDays,
		verified:  index%5 != 3,
		authority: "Certified Medical Examiner",
	})
	plan = append(plan, seedCredential{
		code:      "DOT_PHYSICAL",
		expiresIn: medicalDays,
		issuedAgo: 730 - medicalDays,
		verified:  index%5 != 3,
	})

	mvrDays := int64(200)
	if index%4 == 2 {
		mvrDays = 9
	}
	plan = append(plan, seedCredential{
		code:      "MVR",
		expiresIn: mvrDays,
		issuedAgo: 365 - mvrDays,
		verified:  true,
	})

	if profile.Endorsement.RequiresHazmatExpiry() {
		plan = append(plan, seedCredential{
			code:       "HAZMAT",
			profileSet: true,
			verified:   true,
		})
	}
	if index%3 == 0 {
		plan = append(plan, seedCredential{
			code:      "TWIC",
			expiresIn: 900 - int64(index)*40,
			issuedAgo: 300,
			number:    fmt.Sprintf("TWIC%06d", 240000+index*17),
			authority: "TSA",
			verified:  true,
		})
	}
	if index%2 == 1 {
		plan = append(plan, seedCredential{
			code:      "FORKLIFT",
			expiresIn: 60 + int64(index)*11,
			issuedAgo: 1000,
			authority: "Warehouse Safety Institute",
			notes:     "Class IV and V powered industrial trucks",
		})
	}
	if profile.Endorsement == worker.EndorsementTypeTanker ||
		profile.Endorsement == worker.EndorsementTypeTankerHazmat {
		plan = append(plan, seedCredential{code: "TANKER", noExpiry: true, verified: true})
	}
	if profile.Endorsement == worker.EndorsementTypeDoubleTriple {
		plan = append(plan, seedCredential{code: "DOUBLES", noExpiry: true, verified: true})
	}
	return plan
}

func (s *WorkerCredentialSeed) seedWorker(
	ctx context.Context,
	tx bun.Tx,
	sc *seedhelpers.SeedContext,
	refs *credentialSeedRefs,
	wrk *worker.Worker,
	index int,
) error {
	for _, item := range s.plan(wrk, index) {
		typ, ok := refs.types[item.code]
		if !ok {
			continue
		}

		entity := &worker.WorkerCredential{
			OrganizationID:   refs.orgID,
			BusinessUnitID:   refs.buID,
			WorkerID:         wrk.ID,
			CredentialTypeID: typ.ID,
			Status:           worker.CredentialStatusActive,
			Number:           item.number,
			IssuingAuthority: item.authority,
			Notes:            item.notes,
		}

		switch {
		case item.profileSet:
			entity.ExpiresAt = typ.ProfileField.Expiry(wrk.Profile)
			if entity.ExpiresAt == nil {
				continue
			}
		case item.noExpiry:
			entity.ExpiresAt = nil
		default:
			expires := refs.now + item.expiresIn*credentialSeedDay
			entity.ExpiresAt = &expires
		}
		if item.issuedAgo > 0 {
			issued := refs.now - item.issuedAgo*credentialSeedDay
			entity.IssuedAt = &issued
		}
		if item.verified {
			verifiedAt := refs.now - 3*credentialSeedDay
			entity.VerifiedAt = &verifiedAt
			entity.VerifiedByID = refs.adminID
		}

		result, err := tx.NewInsert().Model(entity).On("CONFLICT DO NOTHING").Exec(ctx)
		if err != nil {
			return err
		}
		if inserted, _ := result.RowsAffected(); inserted == 0 {
			continue
		}
		if err := sc.TrackCreated(ctx, "worker_credentials", entity.ID, s.Name()); err != nil {
			return err
		}
		if !typ.ProfileField.IsSet() || item.profileSet {
			continue
		}
		if err := s.mirror(ctx, tx, refs, wrk, typ.ProfileField, entity); err != nil {
			return err
		}
	}
	return nil
}

func (s *WorkerCredentialSeed) mirror(
	ctx context.Context,
	tx bun.Tx,
	refs *credentialSeedRefs,
	wrk *worker.Worker,
	field worker.CredentialProfileField,
	entity *worker.WorkerCredential,
) error {
	cols := buncolgen.WorkerProfileColumns
	var column buncolgen.Column
	switch field {
	case worker.CredentialProfileFieldMedicalCardExpiry:
		column = cols.MedicalCardExpiry
	case worker.CredentialProfileFieldPhysicalDueDate:
		column = cols.PhysicalDueDate
	case worker.CredentialProfileFieldMVRDueDate:
		column = cols.MVRDueDate
	case worker.CredentialProfileFieldTWICExpiry:
		column = cols.TWICExpiry
	case worker.CredentialProfileFieldHazmatExpiry:
		column = cols.HazmatExpiry
	case worker.CredentialProfileFieldLicenseExpiry, worker.CredentialProfileFieldNone:
		return nil
	}
	_, err := tx.NewUpdate().
		Model((*worker.WorkerProfile)(nil)).
		Set(column.Set(), entity.ExpiresAt).
		Where(cols.WorkerID.Eq(), wrk.ID).
		Where(cols.OrganizationID.Eq(), refs.orgID).
		Where(cols.BusinessUnitID.Eq(), refs.buID).
		Exec(ctx)
	return err
}

func (s *WorkerCredentialSeed) Down(ctx context.Context, tx bun.Tx) error {
	return seedhelpers.RunInTransaction(
		ctx,
		tx,
		s.Name(),
		nil,
		func(ctx context.Context, tx bun.Tx, sc *seedhelpers.SeedContext) error {
			return seedhelpers.DeleteTrackedEntities(ctx, tx, s.Name(), sc)
		},
	)
}

func (s *WorkerCredentialSeed) CanRollback() bool {
	return true
}
