package directoryaccounts

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/identity/app/userservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/mailtext"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/emoss08/trenova/shared/tokenutils"
	"go.uber.org/fx"
)

const (
	metadataDirectoryID = "scimDirectoryId"
	metadataTokenID     = "scimTokenId"
	usernameSuffixBytes = 3
)

type Lifecycle interface {
	ExitOrganization(ctx context.Context, req *userservice.OrganizationExit) error
	EndUserAccess(ctx context.Context, tenantInfo pagination.TenantInfo, userID pulid.ID) error
	PublishUserChange(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		user *tenant.User,
		action string,
	)
}

type Params struct {
	fx.In

	Users     tenantports.UserRepository
	Validator *userservice.Validator
	Lifecycle Lifecycle
	Auditor   permissionports.SecurityAuditor
}

type Service struct {
	users     tenantports.UserRepository
	validator *userservice.Validator
	lifecycle Lifecycle
	auditor   permissionports.SecurityAuditor
}

func New(p Params) *Service {
	return &Service{
		users:     p.Users,
		validator: p.Validator,
		lifecycle: p.Lifecycle,
		auditor:   p.Auditor,
	}
}

type Actor struct {
	DirectoryID    pulid.ID
	TokenID        pulid.ID
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
}

func (a *Actor) TenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: a.OrganizationID, BuID: a.BusinessUnitID}
}

func (a *Actor) Metadata() map[string]any {
	return map[string]any{
		metadataDirectoryID: a.DirectoryID.String(),
		metadataTokenID:     a.TokenID.String(),
	}
}

type Profile struct {
	Name         string
	EmailAddress string
	JobTitle     string
	PhoneNumber  string
	Timezone     string
	UsernameHint string
}

func (p *Profile) normalized() Profile {
	return Profile{
		Name:         strings.TrimSpace(p.Name),
		EmailAddress: mailtext.NormalizeEmailAddress(p.EmailAddress),
		JobTitle:     strings.TrimSpace(p.JobTitle),
		PhoneNumber:  strings.TrimSpace(p.PhoneNumber),
		Timezone:     strings.TrimSpace(p.Timezone),
		UsernameHint: strings.TrimSpace(p.UsernameHint),
	}
}

type CreateRequest struct {
	Actor   Actor
	Profile Profile
	Active  bool
}

func (s *Service) Create(ctx context.Context, req *CreateRequest) (*tenant.User, error) {
	profile := req.Profile.normalized()
	entity := &tenant.User{
		BusinessUnitID:        req.Actor.BusinessUnitID,
		CurrentOrganizationID: req.Actor.OrganizationID,
		Name:                  profile.Name,
		EmailAddress:          profile.EmailAddress,
		JobTitle:              profile.JobTitle,
		PhoneNumber:           profile.PhoneNumber,
		Timezone:              profile.Timezone,
		Status:                domaintypes.StatusActive,
		TimeFormat:            domaintypes.TimeFormat12Hour,
		Locale:                string(i18n.Default),
		AvatarStyle:           tenant.AvatarStyleAuto,
		AvailabilityStatus:    tenant.AvailabilityAvailable,
	}
	if !req.Active {
		entity.Status = domaintypes.StatusInactive
	}

	hint := profile.UsernameHint
	if hint == "" {
		hint = profile.EmailAddress
	}
	username, err := s.availableUsername(ctx, hint)
	if err != nil {
		return nil, err
	}
	entity.Username = username

	if multiErr := s.validator.ValidateCreate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	secret, _, err := tokenutils.New()
	if err != nil {
		return nil, err
	}
	if entity.Password, err = entity.GeneratePassword(secret); err != nil {
		return nil, err
	}

	created, err := s.users.Create(ctx, tenantports.CreateUserRequest{
		User: entity,
		Membership: &tenant.OrganizationMembership{
			BusinessUnitID: req.Actor.BusinessUnitID,
			OrganizationID: req.Actor.OrganizationID,
			IsDefault:      true,
		},
	})
	if err != nil {
		return nil, err
	}

	actor := req.Actor
	kernelports.AfterCommit(ctx, func(committed context.Context) {
		s.recordCreation(committed, &actor, created)
		s.publish(committed, &actor, created, "created")
	})

	return created, nil
}

func (s *Service) availableUsername(ctx context.Context, hint string) (string, error) {
	suffix, err := tokenutils.RandomHex(usernameSuffixBytes)
	if err != nil {
		return "", err
	}

	for _, candidate := range tenant.UsernameCandidates(tenant.UsernameBase(hint), suffix) {
		conflicts, conflictErr := s.users.FindIdentityConflicts(
			ctx,
			tenantports.UserIdentityConflictsRequest{Username: candidate},
		)
		if conflictErr != nil {
			return "", conflictErr
		}
		if !conflicts.Username {
			return candidate, nil
		}
	}

	return "", tenant.NewDuplicateUsernameError()
}

type UpdateRequest struct {
	Actor   Actor
	UserID  pulid.ID
	Profile Profile
}

type UpdateResult struct {
	User     *tenant.User
	Changed  bool
	Withheld bool
}

func (s *Service) Update(ctx context.Context, req *UpdateRequest) (*UpdateResult, error) {
	user, err := s.member(ctx, &req.Actor, req.UserID)
	if err != nil {
		return nil, err
	}

	original := *user
	applyProfile(user, req.Profile.normalized())
	columns, access := changedColumns(&original, user)
	if len(columns) == 0 {
		return &UpdateResult{User: &original}, nil
	}

	memberships, err := s.users.ListOrganizationMemberships(
		ctx,
		req.UserID,
		req.Actor.BusinessUnitID,
	)
	if err != nil {
		return nil, err
	}
	if !solelyIn(&original, memberships, req.Actor.OrganizationID) {
		return &UpdateResult{User: &original, Withheld: true}, nil
	}

	if multiErr := s.validator.ValidateUpdate(ctx, user); multiErr != nil {
		return nil, multiErr
	}

	updated, err := s.users.UpdateColumns(ctx, tenantports.UpdateUserColumnsRequest{
		Entity:  user,
		Columns: columns,
	})
	if err != nil {
		return nil, err
	}

	s.recordChange(ctx, &req.Actor, &permissionports.SecurityChange{
		Resource:   permission.ResourceUser,
		ResourceID: updated.ID.String(),
		Operation:  permission.OpUpdate,
		Before:     &original,
		After:      updated,
		Comment:    "User updated by directory provisioning",
		Metadata:   map[string]any{"changedColumns": columns},
	})
	if len(access) > 0 {
		s.recordAccessChange(ctx, &req.Actor, &original, updated, access)
	}
	s.publish(ctx, &req.Actor, updated, "updated")

	return &UpdateResult{User: updated, Changed: true}, nil
}

func applyProfile(user *tenant.User, profile Profile) {
	if profile.Name != "" {
		user.Name = profile.Name
	}
	if profile.EmailAddress != "" {
		user.EmailAddress = profile.EmailAddress
	}
	user.JobTitle = profile.JobTitle
	user.PhoneNumber = profile.PhoneNumber
	if timeutils.IsValidLocation(profile.Timezone) {
		user.Timezone = profile.Timezone
	}
}

func solelyIn(
	user *tenant.User,
	memberships []*tenant.OrganizationMembership,
	organizationID pulid.ID,
) bool {
	if len(memberships) == 0 {
		return user.CurrentOrganizationID == organizationID
	}

	return coveredBy(memberships, organizationID)
}

func coveredBy(memberships []*tenant.OrganizationMembership, organizationID pulid.ID) bool {
	if len(memberships) == 0 {
		return false
	}
	for _, membership := range memberships {
		if membership.OrganizationID != organizationID {
			return false
		}
	}

	return true
}

func (s *Service) member(ctx context.Context, actor *Actor, userID pulid.ID) (*tenant.User, error) {
	return s.users.GetTenantMember(ctx, tenantports.GetTenantMemberRequest{
		UserID:     userID,
		TenantInfo: actor.TenantInfo(),
		Now:        timeutils.NowUnix(),
	})
}

func (s *Service) setStatus(
	ctx context.Context,
	actor *Actor,
	user *tenant.User,
	status domaintypes.Status,
) (*tenant.User, error) {
	original := *user
	user.Status = status

	updated, err := s.users.UpdateColumns(ctx, tenantports.UpdateUserColumnsRequest{
		Entity:  user,
		Columns: []string{columnName(statusColumn())},
	})
	if err != nil {
		return nil, err
	}

	s.recordAccessChange(ctx, actor, &original, updated, []string{columnName(statusColumn())})
	s.publish(ctx, actor, updated, "updated")

	return updated, nil
}

func isMissing(err error) bool {
	return errortypes.IsNotFoundError(err)
}
