package invoiceshareservice

import (
	"context"
	"fmt"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	servicesports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const systemUsername = "system"

type shareInput struct {
	userIDs []pulid.ID
	note    string
	tab     invoice.ShareTab
}

func validateShareInput(input *shareInput, actorID pulid.ID) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()

	switch {
	case len(input.userIDs) == 0:
		multiErr.Add(
			"userIds",
			errortypes.ErrRequired,
			"Choose at least one teammate to share with",
		)
	case len(input.userIDs) > invoice.MaxShareRecipients:
		multiErr.Add(
			"userIds",
			errortypes.ErrInvalid,
			"An invoice can be shared with at most {0} teammates at a time",
			invoice.MaxShareRecipients,
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
				"You cannot share an invoice with yourself",
			)
		}
	}

	if !input.tab.IsValid() {
		multiErr.Add("tab", errortypes.ErrInvalid, "Invoice tab is invalid")
	}
	if utf8.RuneCountInString(input.note) > invoice.MaxShareNoteLength {
		multiErr.Add(
			"note",
			errortypes.ErrInvalid,
			"Note must be {0} characters or fewer",
			invoice.MaxShareNoteLength,
		)
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func (s *Service) resolveRecipients(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	userIDs []pulid.ID,
) ([]*tenant.User, error) {
	users, err := s.userRepo.GetByIDs(ctx, repositories.GetUsersByIDsRequest{
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

		allowed, checkErr := s.canViewInvoices(ctx, tenantInfo, user)
		if checkErr != nil {
			return nil, checkErr
		}
		if !allowed {
			multiErr.Add(
				field,
				errortypes.ErrInvalidOperation,
				"{0} cannot view invoices, so the link would not open for them",
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

func (s *Service) canViewInvoices(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	user *tenant.User,
) (bool, error) {
	result, err := s.permissions.Check(
		authctx.WithoutSessionRoleActivation(ctx),
		&servicesports.PermissionCheckRequest{
			PrincipalType:  servicesports.PrincipalTypeUser,
			PrincipalID:    user.ID,
			UserID:         user.ID,
			OrganizationID: tenantInfo.OrgID,
			BusinessUnitID: tenantInfo.BuID,
			Resource:       permission.ResourceInvoice.String(),
			Operation:      permission.OpRead,
		},
	)
	if err != nil {
		return false, err
	}

	return result.Allowed, nil
}

func isShareableUser(user *tenant.User) bool {
	return user.Status == domaintypes.StatusActive &&
		!user.IsLocked &&
		user.Username != systemUsername
}
