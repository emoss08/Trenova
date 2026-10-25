package fixtures

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"

	"github.com/emoss08/keel"
	"github.com/emoss08/keel/seed"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

const credentialSeedDay = int64(86400)

type WorkerCredentialSeed struct {
}

// WorkerCredentialSeed gives every seeded driver a realistic credential file:
// the system catalog, a row per mirrored profile column, and a spread of
// healthy, expiring and expired optional credentials so the Credentials tab,
// the expiry forecast and the nightly sweep all have something to show.
//
// Depends on:
//   - Worker: the drivers whose profiles are mirrored
var WorkerCredential = Seeds.Add(seed.Seed{
	Name:        "worker_credential",
	Kind:        seed.Fixture,
	Description: "Seeds credential types and per-worker credentials mirrored from worker profiles",
	After:       []*seed.Seed{Worker},
	Run:         (&WorkerCredentialSeed{}).Run,
})

type credentialSeedRefs struct {
	orgID   pulid.ID
	buID    pulid.ID
	adminID pulid.ID
	now     int64
	types   map[string]*worker.WorkerCredentialType
}

func (s *WorkerCredentialSeed) Run(ctx context.Context, sr *seed.Run) error {
	db := sr.DB
	org, err := defaultOrganization(ctx, sr)
	if err != nil {
		return err
	}
	admin, err := userByUsername(ctx, db, "admin")
	if err != nil {
		return fmt.Errorf("get admin user: %w", err)
	}
	refs := &credentialSeedRefs{
		orgID:   org.ID,
		buID:    org.BusinessUnitID,
		adminID: admin.ID,
		now:     timeutils.NowUnix(),
	}

	if err = s.ensureTypes(ctx, db, refs); err != nil {
		return fmt.Errorf("ensure credential types: %w", err)
	}

	workers, err := s.loadWorkers(ctx, db, refs)
	if err != nil {
		return fmt.Errorf("load workers: %w", err)
	}
	for i, wrk := range workers {
		if err = s.seedWorker(ctx, db, sr, refs, wrk, i); err != nil {
			return fmt.Errorf("seed credentials for worker %s: %w", wrk.ID, err)
		}
	}

	// Compliance is derived, never asserted. The worker seed sets every
	// profile Compliant before any credential exists, which produced
	// drivers reading "no medical card on file" and "Compliant" in the
	// same row — a contradiction the assistant resolved in favour of
	// the friendlier half, and which would hide exactly the driver a
	// compliance question is asked to surface.
	for _, wrk := range workers {
		if err = s.deriveCompliance(ctx, db, refs, wrk); err != nil {
			return fmt.Errorf("derive compliance for worker %s: %w", wrk.ID, err)
		}
	}

	return nil
}

func (s *WorkerCredentialSeed) ensureTypes(
	ctx context.Context,
	db *keel.DB,
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
	if _, err := workerdb.WorkerCredentialTypes.CreateOrSkip(ctx, db, rows); err != nil {
		return err
	}

	existing, err := workerdb.WorkerCredentialTypes.AllTenants().Where(
		workerdb.WorkerCredentialTypes.OrganizationID.Eq(refs.orgID),
		workerdb.WorkerCredentialTypes.BusinessUnitID.Eq(refs.buID),
	).All(ctx, db)
	if err != nil {
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
	db *keel.DB,
	refs *credentialSeedRefs,
) ([]*worker.Worker, error) {
	workers, err := workerdb.Workers.AllTenants().Load(workerdb.Workers.Profile()).Where(
		workerdb.Workers.OrganizationID.Eq(refs.orgID),
		workerdb.Workers.BusinessUnitID.Eq(refs.buID),
	).Order(workerdb.Workers.CreatedAt.Asc()).All(ctx, db)
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
	plan := make([]seedCredential, 0, 11)
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
	plan = append(plan,
		seedCredential{code: "ROAD_TEST", noExpiry: true, issuedAgo: 1400, verified: true},
		seedCredential{
			code:      "ANNUAL_REVIEW",
			expiresIn: 250 - int64(index)*7,
			issuedAgo: 115 + int64(index)*7,
			verified:  true,
		},
		seedCredential{
			code:      "VIOLATION_CERT",
			expiresIn: 250 - int64(index)*7,
			issuedAgo: 115 + int64(index)*7,
			verified:  true,
		},
	)

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
	db *keel.DB,
	sr *seed.Run,
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

		inserted, err := workerdb.WorkerCredentials.CreateOrSkip(
			ctx,
			db,
			[]*worker.WorkerCredential{entity},
		)
		if err != nil {
			return err
		}
		if inserted == 0 {
			continue
		}
		if !typ.ProfileField.IsSet() || item.profileSet {
			continue
		}
		if err := s.mirror(ctx, db, refs, wrk, typ.ProfileField, entity); err != nil {
			return err
		}
	}
	return nil
}

// deriveCompliance replaces the seeded assumption with the domain's own answer.
//
// BuildCredentialSummary is the same roll-up the application runs: a required
// slot with no active credential is Missing, Missing and Expired both block,
// and a worker with no required types is compliant by construction. Calling it
// here rather than reimplementing the rule is the point — seed data that
// disagrees with the domain is worse than no seed data, because it looks real.
//
// Only compliance_status is touched. is_qualified is a separate determination
// with no derivation anywhere in the domain — somebody decides it — so deriving
// it from credentials here would invent a rule the application does not have.
func (s *WorkerCredentialSeed) deriveCompliance(
	ctx context.Context,
	db *keel.DB,
	refs *credentialSeedRefs,
	wrk *worker.Worker,
) error {
	credentials, err := workerdb.WorkerCredentials.AllTenants().Where(
		workerdb.WorkerCredentials.WorkerID.Eq(wrk.ID),
		workerdb.WorkerCredentials.OrganizationID.Eq(refs.orgID),
		workerdb.WorkerCredentials.BusinessUnitID.Eq(refs.buID),
	).All(ctx, db)
	if err != nil {
		return fmt.Errorf("load credentials: %w", err)
	}

	types := make([]*worker.WorkerCredentialType, 0, len(refs.types))
	for _, typ := range refs.types {
		types = append(types, typ)
	}

	summary := worker.BuildCredentialSummary(wrk, types, credentials, refs.now)

	wp := workerdb.WorkerProfiles
	_, err = wp.AllTenants().Where(
		wp.WorkerID.Eq(wrk.ID),
		wp.OrganizationID.Eq(refs.orgID),
		wp.BusinessUnitID.Eq(refs.buID),
	).Update(ctx, db, wp.ComplianceStatus.To(summary.ComplianceStatus))

	return err
}

func (s *WorkerCredentialSeed) mirror(
	ctx context.Context,
	db *keel.DB,
	refs *credentialSeedRefs,
	wrk *worker.Worker,
	field worker.CredentialProfileField,
	entity *worker.WorkerCredential,
) error {
	wp := workerdb.WorkerProfiles
	var column keel.Col
	switch field {
	case worker.CredentialProfileFieldMedicalCardExpiry:
		column = wp.MedicalCardExpiry.Col
	case worker.CredentialProfileFieldPhysicalDueDate:
		column = wp.PhysicalDueDate.Col
	case worker.CredentialProfileFieldMVRDueDate:
		column = wp.MVRDueDate.Col
	case worker.CredentialProfileFieldTWICExpiry:
		column = wp.TWICExpiry.Col
	case worker.CredentialProfileFieldHazmatExpiry:
		column = wp.HazmatExpiry.Col
	case worker.CredentialProfileFieldLicenseExpiry, worker.CredentialProfileFieldNone:
		return nil
	}
	_, err := wp.AllTenants().Where(
		wp.WorkerID.Eq(wrk.ID),
		wp.OrganizationID.Eq(refs.orgID),
		wp.BusinessUnitID.Eq(refs.buID),
	).Update(ctx, db, column.ToExpr(keel.DynLit(entity.ExpiresAt)))
	return err
}
