package scimservice

import (
	"context"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/identity/app/directoryaccounts"
	"github.com/emoss08/trenova/internal/modules/identity/app/orgzone"
	"github.com/emoss08/trenova/internal/modules/identity/domain/iam"
	"github.com/emoss08/trenova/internal/modules/identity/domain/scim"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/scimports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/scimproto"
	"github.com/emoss08/trenova/shared/timeutils"
)

const maxAttributeLength = 255

type desiredUser struct {
	userName   string
	externalID string
	active     bool
	profile    directoryaccounts.Profile
}

func desiredFrom(resource *scimproto.User, requireEmail bool) (*desiredUser, error) {
	userName := strings.TrimSpace(resource.UserName)
	if userName == "" {
		return nil, scimproto.InvalidValue("userName is required")
	}
	if len(userName) > maxAttributeLength {
		return nil, scimproto.InvalidValue("userName cannot be longer than 255 characters")
	}

	externalID := strings.TrimSpace(resource.ExternalID)
	if len(externalID) > maxAttributeLength {
		return nil, scimproto.InvalidValue("externalId cannot be longer than 255 characters")
	}

	email := resource.PrimaryEmail()
	if email == "" && strings.Contains(userName, "@") {
		email = userName
	}
	if email == "" && requireEmail {
		return nil, scimproto.InvalidValue("The user needs an email address or an email userName")
	}

	return &desiredUser{
		userName:   userName,
		externalID: externalID,
		active:     resource.IsActive(),
		profile: directoryaccounts.Profile{
			Name:         resource.FullName(),
			EmailAddress: email,
			JobTitle:     strings.TrimSpace(resource.Title),
			PhoneNumber:  resource.PrimaryPhone(),
			Timezone:     strings.TrimSpace(resource.Timezone),
			UsernameHint: email,
		},
	}, nil
}

func (s *Service) ListUsers(
	ctx context.Context,
	req *scimports.ListRequest,
) (*scimports.UserPage, error) {
	filter, err := scimproto.ParseFilter(req.Filter, scimports.UserFilters()...)
	if err != nil {
		return nil, err
	}

	start, size := pageBounds(req.StartIndex, req.Count)
	records, total, err := s.repo.ListUsers(scopedTo(ctx, req.Directory), &scimports.ListQuery{
		DirectoryID: req.Directory.ID,
		Filter:      filter,
		Offset:      start - 1,
		Limit:       max(size, 1),
	})
	if err != nil {
		return nil, err
	}
	if size == 0 {
		records = nil
	}

	users := make([]*scimproto.User, 0, len(records))
	for _, record := range records {
		users = append(users, userResource(record))
	}

	return &scimports.UserPage{Users: users, Total: total, StartIndex: start}, nil
}

func (s *Service) GetUser(
	ctx context.Context,
	dir *scimports.Directory,
	userID pulid.ID,
) (*scimproto.User, error) {
	record, err := s.repo.GetUser(scopedTo(ctx, dir), dir.ID, userID)
	if err != nil {
		return nil, err
	}

	return userResource(record), nil
}

func (s *Service) CreateUser(
	ctx context.Context,
	dir *scimports.Directory,
	resource *scimproto.User,
) (*scimproto.User, error) {
	desired, err := desiredFrom(resource, true)
	if err != nil {
		return nil, err
	}

	ctx = scopedTo(ctx, dir)
	if !timeutils.IsValidLocation(desired.profile.Timezone) {
		zone, zoneErr := orgzone.Location(ctx, s.organizations, dir.TenantInfo())
		if zoneErr != nil {
			return nil, zoneErr
		}
		desired.profile.Timezone = zone.String()
	}

	e := &event{
		action:       iam.ProvisioningActionCreate,
		resourceType: resourceTypeUser,
		externalID:   desired.externalID,
	}

	record, linked, err := s.provisionUser(ctx, dir, desired)
	if err != nil {
		s.recordFailure(ctx, dir, e, err)
		return nil, err
	}

	if linked {
		if record, err = s.reconcileAccount(ctx, dir, record, desired, true); err != nil {
			e.resourceID = record.Link.UserID
			s.recordFailure(ctx, dir, e, err)
			return nil, err
		}
	}

	e.resourceID = record.Link.UserID
	e.status = statusSucceeded
	e.comment = "SCIM user provisioned"
	if linked {
		e.comment = "SCIM user linked to an existing member"
	}
	e.after = record.Link
	s.record(ctx, dir, e)

	return userResource(record), nil
}

func (s *Service) provisionUser(
	ctx context.Context,
	dir *scimports.Directory,
	desired *desiredUser,
) (record *scimports.UserRecord, linked bool, err error) {
	err = s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		existing, findErr := s.repo.FindMember(txCtx, desired.profile.EmailAddress)
		switch {
		case findErr == nil:
			if existing.Username == tenant.SystemUsername {
				return scimproto.Uniqueness("A user with this email address already exists")
			}
			linked = true
			link := newLink(dir, existing.ID, desired, false)
			if createErr := s.repo.CreateUser(txCtx, link); createErr != nil {
				return createErr
			}
			record = &scimports.UserRecord{Link: link, User: existing}
			return nil
		case errortypes.IsNotFoundError(findErr):
			created, createErr := s.accounts.Create(txCtx, &directoryaccounts.CreateRequest{
				Actor:   actorOf(dir),
				Profile: desired.profile,
				Active:  desired.active,
			})
			if createErr != nil {
				return createErr
			}
			link := newLink(dir, created.ID, desired, true)
			if createErr = s.repo.CreateUser(txCtx, link); createErr != nil {
				return createErr
			}
			record = &scimports.UserRecord{Link: link, User: created}
			return nil
		default:
			return findErr
		}
	})

	return record, linked, err
}

func newLink(
	dir *scimports.Directory,
	userID pulid.ID,
	desired *desiredUser,
	provisioned bool,
) *scim.User {
	return &scim.User{
		OrganizationID: dir.OrganizationID,
		BusinessUnitID: dir.BusinessUnitID,
		DirectoryID:    dir.ID,
		UserID:         userID,
		UserName:       desired.userName,
		ExternalID:     desired.externalID,
		Active:         desired.active,
		Provisioned:    provisioned,
	}
}

func (s *Service) ReplaceUser(
	ctx context.Context,
	dir *scimports.Directory,
	userID pulid.ID,
	resource *scimproto.User,
) (*scimproto.User, error) {
	desired, err := desiredFrom(resource, false)
	if err != nil {
		return nil, err
	}

	return s.updateUser(
		scopedTo(ctx, dir),
		dir,
		userID,
		func(*scimproto.User) (*desiredUser, error) {
			return desired, nil
		},
	)
}

func (s *Service) PatchUser(
	ctx context.Context,
	dir *scimports.Directory,
	userID pulid.ID,
	operations []scimproto.PatchOperation,
) (*scimproto.User, error) {
	return s.updateUser(
		scopedTo(ctx, dir),
		dir,
		userID,
		func(current *scimproto.User) (*desiredUser, error) {
			if err := scimproto.ApplyUserPatch(current, operations); err != nil {
				return nil, err
			}
			return desiredFrom(current, false)
		},
	)
}

func (s *Service) updateUser(
	ctx context.Context,
	dir *scimports.Directory,
	userID pulid.ID,
	desire func(current *scimproto.User) (*desiredUser, error),
) (*scimproto.User, error) {
	record, err := s.repo.GetUser(ctx, dir.ID, userID)
	if err != nil {
		return nil, err
	}

	desired, err := desire(userResource(record))
	if err != nil {
		return nil, err
	}

	before := *record.Link
	e := &event{
		action:       iam.ProvisioningActionUpdate,
		resourceType: resourceTypeUser,
		resourceID:   userID,
		externalID:   desired.externalID,
		before:       &before,
	}
	if before.Active && !desired.active {
		e.action = iam.ProvisioningActionDeactivate
	}

	record, err = s.reconcileAccount(ctx, dir, record, desired, false)
	if err != nil {
		s.recordFailure(ctx, dir, e, err)
		return nil, err
	}

	e.status = statusSucceeded
	e.comment = "SCIM user updated"
	if e.action == iam.ProvisioningActionDeactivate {
		e.comment = "SCIM user deactivated"
	}
	e.after = record.Link
	s.record(ctx, dir, e)

	return userResource(record), nil
}

func (s *Service) reconcileAccount(
	ctx context.Context,
	dir *scimports.Directory,
	record *scimports.UserRecord,
	desired *desiredUser,
	linking bool,
) (*scimports.UserRecord, error) {
	link := record.Link
	actor := actorOf(dir)
	access := &directoryaccounts.AccessRequest{Actor: actor, UserID: link.UserID}
	wasActive := link.Active
	if linking {
		wasActive = !desired.active
	}

	if desired.active && !wasActive {
		if err := s.accounts.Restore(ctx, access); err != nil {
			return record, err
		}
		reloaded, err := s.repo.GetUser(ctx, dir.ID, link.UserID)
		if err != nil {
			return record, err
		}
		record = reloaded
	}

	if record.User != nil {
		result, err := s.accounts.Update(ctx, &directoryaccounts.UpdateRequest{
			Actor:   actor,
			UserID:  link.UserID,
			Profile: desired.profile,
		})
		if err != nil {
			return record, err
		}
		if result.Withheld {
			s.record(ctx, dir, &event{
				action:       iam.ProvisioningActionUpdate,
				resourceType: resourceTypeUser,
				resourceID:   link.UserID,
				externalID:   desired.externalID,
				status:       statusWithheld,
				message:      "The person also belongs to other organizations, so their profile is kept",
			})
		}
		record.User = result.User
	}

	if !desired.active && wasActive {
		if err := s.accounts.Revoke(ctx, access); err != nil {
			return record, err
		}
	}

	activeChanged := record.Link.Active != desired.active
	if activeChanged || record.Link.UserName != desired.userName ||
		record.Link.ExternalID != desired.externalID {
		record.Link.UserName = desired.userName
		record.Link.ExternalID = desired.externalID
		record.Link.Active = desired.active
		if err := s.repo.SaveUser(ctx, record.Link); err != nil {
			return record, err
		}
	}

	if activeChanged || linking {
		if err := s.repo.MarkPending(ctx, dir.ID, []pulid.ID{link.UserID}); err != nil {
			return record, err
		}
		s.requester.RequestReconcile(ctx, dir.TenantInfo(), dir.ID)
	}

	if !desired.active {
		return s.repo.GetUser(ctx, dir.ID, link.UserID)
	}

	return record, nil
}

func (s *Service) DeleteUser(ctx context.Context, dir *scimports.Directory, userID pulid.ID) error {
	ctx = scopedTo(ctx, dir)
	record, err := s.repo.GetUser(ctx, dir.ID, userID)
	if err != nil {
		return err
	}

	e := &event{
		action:       iam.ProvisioningActionDelete,
		resourceType: resourceTypeUser,
		resourceID:   userID,
		externalID:   record.Link.ExternalID,
		before:       record.Link,
	}

	if err = s.accounts.Revoke(ctx, &directoryaccounts.AccessRequest{
		Actor:  actorOf(dir),
		UserID: userID,
	}); err != nil {
		s.recordFailure(ctx, dir, e, err)
		return err
	}
	if err = s.revokeUserGrants(ctx, dir, userID); err != nil {
		s.recordFailure(ctx, dir, e, err)
		return err
	}
	if err = s.repo.DeleteUser(ctx, dir.ID, userID); err != nil {
		s.recordFailure(ctx, dir, e, err)
		return err
	}

	e.status = statusSucceeded
	e.comment = "SCIM user removed from the directory"
	s.record(ctx, dir, e)

	return nil
}

func userResource(record *scimports.UserRecord) *scimproto.User {
	link := record.Link
	resource := &scimproto.User{
		Schemas:    []string{scimproto.SchemaUser},
		ID:         link.UserID.String(),
		ExternalID: link.ExternalID,
		UserName:   link.UserName,
		Active:     scimproto.BoolOf(link.Active),
		Meta: &scimproto.Meta{
			ResourceType: scimproto.ResourceTypeUser,
			Created:      timestamp(link.CreatedAt),
			LastModified: timestamp(link.UpdatedAt),
		},
	}

	user := record.User
	if user == nil {
		return resource
	}

	given, family, _ := strings.Cut(strings.TrimSpace(user.Name), " ")
	resource.Name = &scimproto.Name{
		GivenName:  given,
		FamilyName: strings.TrimSpace(family),
	}
	resource.DisplayName = user.Name
	resource.Title = user.JobTitle
	resource.Timezone = user.Timezone
	resource.Locale = user.Locale
	resource.Emails = []scimproto.MultiValue{{
		Value:   user.EmailAddress,
		Type:    "work",
		Primary: true,
	}}
	if user.PhoneNumber != "" {
		resource.PhoneNumbers = []scimproto.MultiValue{{Value: user.PhoneNumber, Type: "work"}}
	}
	if user.UpdatedAt > link.UpdatedAt {
		resource.Meta.LastModified = timestamp(user.UpdatedAt)
	}

	return resource
}

func timestamp(unix int64) string {
	if unix <= 0 {
		return ""
	}

	return time.Unix(unix, 0).UTC().Format(time.RFC3339)
}
