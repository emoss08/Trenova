package seedaccountservice

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/cloud/seedaccounts/seedaccountport"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

const unusableSecretBytes = 32

var ErrApplyNeedsSideEffects = errors.New(
	"applying the retirement needs the session store, the permission cache and the security auditor",
)

type Params struct {
	fx.In

	Repo            seedaccountport.Repository
	Sessions        repositories.SessionRepository         `optional:"true"`
	PermissionCache repositories.PermissionCacheRepository `optional:"true"`
	Auditor         services.SecurityAuditor               `optional:"true"`
	Logger          *zap.Logger
}

type Service struct {
	repo           seedaccountport.Repository
	sessions       repositories.SessionRepository
	permCache      repositories.PermissionCacheRepository
	auditor        services.SecurityAuditor
	l              *zap.Logger
	now            func() int64
	unusableHash   func() (string, error)
	seededPassword func(hash string) bool
}

type RunRequest struct {
	Apply bool
}

func New(p Params) *Service {
	return &Service{
		repo:           p.Repo,
		sessions:       p.Sessions,
		permCache:      p.PermissionCache,
		auditor:        p.Auditor,
		l:              p.Logger.Named("service.seed-account-retirement"),
		now:            timeutils.NowUnix,
		unusableHash:   unusablePasswordHash,
		seededPassword: matchesSeededPassword,
	}
}

func (s *Service) Run(ctx context.Context, req RunRequest) (*Report, error) {
	if !req.Apply {
		var report *Report
		err := s.repo.InTransaction(
			ctx,
			seedaccountport.TransactionOptions{ReadOnly: true},
			func(ctx context.Context) error {
				var planErr error
				report, planErr = s.plan(ctx, false)
				return planErr
			},
		)
		if err != nil {
			return nil, err
		}

		return report, nil
	}

	if s.sessions == nil || s.permCache == nil || s.auditor == nil {
		return nil, ErrApplyNeedsSideEffects
	}

	var report *Report
	err := s.repo.InTransaction(
		ctx,
		seedaccountport.TransactionOptions{},
		func(ctx context.Context) error {
			planned, planErr := s.plan(ctx, true)
			if planErr != nil {
				return planErr
			}
			if execErr := s.execute(ctx, planned); execErr != nil {
				return execErr
			}
			report = planned
			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	report.Applied = true
	s.revokeAccess(ctx, report)
	s.audit(ctx, report)

	return report, nil
}

func (s *Service) plan(ctx context.Context, forUpdate bool) (*Report, error) {
	system, err := s.repo.FindSystemUser(ctx)
	if err != nil {
		return nil, err
	}

	footprints, err := s.repo.FindUsers(ctx, &seedaccountport.FindUsersRequest{
		Accounts:  seedaccountport.KnownAccounts(),
		SeedName:  seedaccountport.SeedName,
		ForUpdate: forUpdate,
	})
	if err != nil {
		return nil, err
	}

	report := &Report{
		SystemUser:    system,
		Users:         make([]*UserReport, 0, len(footprints)),
		Organizations: make([]*OrganizationReport, 0),
		Errors:        make([]string, 0),
	}
	for _, fp := range footprints {
		if !knownAccount(fp) {
			continue
		}
		report.Users = append(report.Users, classifyUser(fp, s.seededPassword))
	}

	retiring := report.retiring()
	for _, user := range report.Users {
		if user.Action != UserActionRetire {
			continue
		}
		refs, refErr := s.repo.CountReferences(ctx, &seedaccountport.CountReferencesRequest{
			UserID:        user.Footprint.ID,
			ExcludeOwners: retiring,
		})
		if refErr != nil {
			return nil, refErr
		}
		user.Footprint.References = refs
		user.Delete = len(refs) == 0
	}

	if !forUpdate {
		if err = s.inspectOrganizations(ctx, report, report.leaving(), false); err != nil {
			return nil, err
		}
	}

	return report, nil
}

func (s *Service) inspectOrganizations(
	ctx context.Context,
	report *Report,
	exclude []pulid.ID,
	forUpdate bool,
) error {
	orgs, err := s.repo.InspectOrganizations(ctx, &seedaccountport.InspectOrganizationsRequest{
		Organizations: seedaccountport.KnownOrganizations(),
		SeedName:      seedaccountport.SeedName,
		ExcludeUsers:  exclude,
		ExcludeOwners: report.retiring(),
		ForUpdate:     forUpdate,
	})
	if err != nil {
		return err
	}

	report.Organizations = make([]*OrganizationReport, 0, len(orgs))
	for _, org := range orgs {
		if !knownOrganization(org) {
			continue
		}
		report.Organizations = append(
			report.Organizations,
			classifyOrganization(org, report.SystemUser),
		)
	}

	return nil
}

func (s *Service) execute(ctx context.Context, report *Report) error {
	now := s.now()
	var revokedBy pulid.ID
	if report.SystemUser != nil {
		revokedBy = report.SystemUser.ID
	}

	for _, user := range report.Users {
		if user.Action != UserActionRetire {
			continue
		}
		stripped, err := s.repo.StripUser(ctx, &seedaccountport.StripUserRequest{
			UserID:      user.Footprint.ID,
			RevokedByID: revokedBy,
			Now:         now,
		})
		if err != nil {
			return fmt.Errorf("retire %s: %w", user.Footprint.Username, err)
		}
		user.Stripped = stripped
	}

	for _, user := range report.Users {
		if user.Action != UserActionRetire {
			continue
		}
		hash, err := s.unusableHash()
		if err != nil {
			return fmt.Errorf(
				"generate an unusable password for %s: %w",
				user.Footprint.Username,
				err,
			)
		}
		removed, err := s.repo.RemoveUser(ctx, &seedaccountport.RemoveUserRequest{
			UserID:       user.Footprint.ID,
			PasswordHash: hash,
			Now:          now,
		})
		if err != nil {
			return fmt.Errorf("retire %s: %w", user.Footprint.Username, err)
		}
		user.Removed = removed
		user.Delete = removed.Deleted
		user.Footprint.References = removed.References
	}

	if err := s.inspectOrganizations(ctx, report, nil, true); err != nil {
		return err
	}

	for _, org := range report.Organizations {
		if org.Action != OrganizationActionRemove {
			continue
		}
		result, err := s.repo.RemoveOrganization(ctx, org.Footprint)
		if err != nil {
			return fmt.Errorf("remove organization %s: %w", org.Footprint.Name, err)
		}
		org.Removed = result.Deleted
		if !result.Deleted {
			org.Action = OrganizationActionReview
			org.Reasons = append(org.Reasons, "the database kept it: "+result.RetainedReason)
		}
	}

	return nil
}

func (s *Service) revokeAccess(ctx context.Context, report *Report) {
	for _, user := range report.Users {
		if user.Action == UserActionUnproven {
			continue
		}
		if err := s.sessions.DeleteAllForUser(ctx, user.Footprint.ID); err != nil {
			s.fail(report, fmt.Sprintf("revoke the sessions of %s", user.Footprint.Username), err)
		} else {
			user.SessionsRevoked = true
		}

		if user.Stripped == nil {
			continue
		}
		for _, orgID := range strippedOrganizations(user.Stripped) {
			if err := s.permCache.Delete(ctx, user.Footprint.ID, orgID); err != nil {
				s.fail(
					report,
					fmt.Sprintf(
						"clear the cached permissions of %s in %s",
						user.Footprint.Username,
						orgID,
					),
					err,
				)
			}
		}
	}
}

func (s *Service) fail(report *Report, what string, err error) {
	report.Errors = append(report.Errors, what+": "+err.Error())
	s.l.Error("seed account retirement step failed", zap.String("step", what), zap.Error(err))
}

func strippedOrganizations(stripped *seedaccountport.StripUserResult) []pulid.ID {
	seen := make(map[pulid.ID]struct{})
	out := make([]pulid.ID, 0, len(stripped.Memberships))
	add := func(id pulid.ID) {
		if id.IsNil() {
			return
		}
		if _, dup := seen[id]; dup {
			return
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	for _, m := range stripped.Memberships {
		add(m.OrganizationID)
	}
	for _, a := range stripped.RoleAssignments {
		add(a.OrganizationID)
	}

	return out
}

func unusablePasswordHash() (string, error) {
	secret := make([]byte, unusableSecretBytes)
	if _, err := rand.Read(secret); err != nil {
		return "", err
	}

	return new(tenant.User).GeneratePassword(hex.EncodeToString(secret))
}

func matchesSeededPassword(hash string) bool {
	return bcrypt.CompareHashAndPassword(
		[]byte(hash),
		[]byte(seedaccountport.SeededPassword),
	) == nil
}
