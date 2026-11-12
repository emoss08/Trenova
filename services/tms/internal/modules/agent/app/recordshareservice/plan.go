package recordshareservice

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/modules/agent/domain/recordshare"
	"github.com/emoss08/trenova/internal/modules/agent/ports/recordshareports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	systemUsername = "system"
	activeStatus   = domaintypes.StatusActive
)

type shareInput struct {
	userIDs []pulid.ID
	note    string
	path    string
}

type sharePlan struct {
	record     *recordshareports.Record
	sharer     *tenant.User
	recipients []*tenant.User
	shares     []*recordshare.Share
	input      shareInput
}

func (s *Service) planShare(
	ctx context.Context,
	req *recordshareports.ShareRequest,
	actor *permissionports.RequestActor,
) (*sharePlan, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Request is required",
		)
	}
	if !actor.IsUser() || actor.UserID.IsNil() {
		return nil, errortypes.NewValidationError(
			"actor",
			errortypes.ErrInvalidOperation,
			"Only a signed-in user can share a record",
		)
	}

	plan := &sharePlan{input: shareInput{
		userIDs: sliceutils.Dedupe(req.UserIDs),
		note:    strings.TrimSpace(req.Note),
		path:    strings.TrimSpace(req.Path),
	}}
	if err := validateShareInput(&plan.input, actor.UserID); err != nil {
		return nil, err
	}

	var err error
	if plan.record, err = s.record(ctx, req.TenantInfo, req.Kind, req.RecordID, actor); err != nil {
		return nil, err
	}
	if plan.sharer, err = s.userRepo.GetByID(ctx, tenantports.GetUserByIDRequest{
		TenantInfo:   req.TenantInfo,
		LookupUserID: actor.UserID,
	}); err != nil {
		return nil, err
	}
	if plan.recipients, err = s.resolveRecipients(
		ctx,
		req.TenantInfo,
		plan.record,
		plan.input.userIDs,
	); err != nil {
		return nil, err
	}
	if plan.shares, err = buildShares(req.TenantInfo, plan); err != nil {
		return nil, err
	}

	return plan, nil
}

func validateShareInput(input *shareInput, actorID pulid.ID) error {
	multiErr := errortypes.NewMultiError()

	switch {
	case len(input.userIDs) == 0:
		multiErr.Add(
			"userIds",
			errortypes.ErrRequired,
			"Choose at least one teammate to share with",
		)
	case len(input.userIDs) > recordshare.MaxRecipients:
		multiErr.Add(
			"userIds",
			errortypes.ErrInvalid,
			"A record can be shared with at most {0} teammates at a time",
			recordshare.MaxRecipients,
		)
	}

	for i, id := range input.userIDs {
		if id.IsNil() {
			multiErr.Add(
				fmt.Sprintf("userIds[%d]", i),
				errortypes.ErrRequired,
				"Recipient is required",
			)

			continue
		}
		if id == actorID {
			multiErr.Add(
				fmt.Sprintf("userIds[%d]", i),
				errortypes.ErrInvalid,
				"You cannot share something with yourself",
			)
		}
	}

	if !recordshare.ValidPath(input.path) {
		multiErr.Add("path", errortypes.ErrInvalid, "The link must be an address inside the app")
	}
	if utf8.RuneCountInString(input.note) > recordshare.MaxNoteLength {
		multiErr.Add(
			"note",
			errortypes.ErrInvalid,
			"Note must be {0} characters or fewer",
			recordshare.MaxNoteLength,
		)
	}

	return multiErr.Err()
}

func buildShares(tenantInfo pagination.TenantInfo, plan *sharePlan) ([]*recordshare.Share, error) {
	now := timeutils.NowUnix()
	shares := make([]*recordshare.Share, 0, len(plan.recipients))
	multiErr := errortypes.NewMultiError()

	for _, recipient := range plan.recipients {
		share := &recordshare.Share{
			OrganizationID: tenantInfo.OrgID,
			BusinessUnitID: tenantInfo.BuID,
			RecordKind:     plan.record.Kind,
			RecordID:       plan.record.ID,
			SharedWithID:   recipient.ID,
			SharedByID:     plan.sharer.ID,
			Note:           plan.input.note,
			Path:           plan.input.path,
			ShareCount:     1,
			FirstSharedAt:  now,
			LastSharedAt:   now,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		share.Validate(multiErr)
		shares = append(shares, share)
	}

	if multiErr.HasErrors() {
		return nil, multiErr.Err()
	}

	return shares, nil
}

func (s *Service) resolveRecipients(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	record *recordshareports.Record,
	userIDs []pulid.ID,
) ([]*tenant.User, error) {
	users, err := s.userRepo.GetByIDs(ctx, tenantports.GetUsersByIDsRequest{
		TenantInfo: tenantInfo,
		UserIDs:    userIDs,
	})
	if err != nil {
		return nil, err
	}

	byID := make(map[pulid.ID]*tenant.User, len(users))
	for _, user := range users {
		byID[user.ID] = user
	}

	recipients := make([]*tenant.User, 0, len(userIDs))
	multiErr := errortypes.NewMultiError()

	for i, id := range userIDs {
		field := fmt.Sprintf("userIds[%d]", i)
		user, ok := byID[id]
		if !ok || !isShareableUser(user) {
			multiErr.Add(
				field,
				errortypes.ErrInvalid,
				"This user is not an active member of your organization",
			)

			continue
		}

		allowed, checkErr := s.canOpen(ctx, tenantInfo, record.Kind.Resource(), user)
		if checkErr != nil {
			return nil, checkErr
		}
		if !allowed {
			multiErr.Add(
				field,
				errortypes.ErrInvalidOperation,
				"{0} cannot view this kind of record, so the link would not open for them",
				user.Name,
			)

			continue
		}

		recipients = append(recipients, user)
	}

	if multiErr.HasErrors() {
		return nil, multiErr
	}

	return recipients, nil
}

// canOpen says whether a person's own role lets them read this kind of
// record. Sharing sends a link and nothing more, so it is only offered to
// someone the link will open for.
func (s *Service) canOpen(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	resource permission.Resource,
	user *tenant.User,
) (bool, error) {
	result, err := s.permissions.Check(
		authctx.WithoutSessionRoleActivation(ctx),
		&permissionports.PermissionCheckRequest{
			PrincipalType:  permissionports.PrincipalTypeUser,
			PrincipalID:    user.ID,
			UserID:         user.ID,
			OrganizationID: tenantInfo.OrgID,
			BusinessUnitID: tenantInfo.BuID,
			Resource:       resource.String(),
			Operation:      permission.OpRead,
		},
	)
	if err != nil {
		return false, err
	}

	return result.Allowed, nil
}
