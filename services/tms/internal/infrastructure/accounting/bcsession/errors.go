package bcsession

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/shared/businesscentralerrors"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	maxErrorMessage      = 2000
	defaultRateLimitWait = 30 * time.Second
)

var ErrDuplicateName = errors.New(
	"businesscentral: a record with this name already exists in Business Central",
)

func ClassifyError(err error) accountingsync.ErrorCategory {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, ErrNotConfigured), errors.Is(err, ErrNoRedirect),
		errors.Is(err, ErrNoAccess):
		return accountingsync.ErrorCategoryConfiguration
	case businesscentralerrors.IsInvalidGrant(err):
		return accountingsync.ErrorCategoryRevoked
	case businesscentralerrors.IsRateLimited(err):
		return accountingsync.ErrorCategoryRateLimited
	case businesscentralerrors.IsInvalidClient(err), businesscentralerrors.IsAuth(err):
		return accountingsync.ErrorCategoryUnauthorized
	case businesscentralerrors.IsForbidden(err):
		return accountingsync.ErrorCategoryConfiguration
	case businesscentralerrors.IsTransient(err), errors.Is(err, context.DeadlineExceeded):
		return accountingsync.ErrorCategoryTransient
	default:
		return accountingsync.ErrorCategoryUnknown
	}
}

type documentFault struct {
	matches    func(err error) bool
	category   accountingsync.SyncErrorCategory
	resolution string
}

func isConfigurationFault(err error) bool {
	return errors.Is(err, ErrNotConfigured) || errors.Is(err, ErrNoRedirect) ||
		errors.Is(err, ErrNoAccess)
}

func isAuthFault(err error) bool {
	return businesscentralerrors.IsInvalidGrant(err) ||
		businesscentralerrors.IsInvalidClient(err) ||
		businesscentralerrors.IsAuth(err)
}

func isTransientFault(err error) bool {
	return businesscentralerrors.IsTransient(err) || errors.Is(err, context.DeadlineExceeded)
}

func isDuplicateName(err error) bool {
	return errors.Is(err, ErrDuplicateName)
}

func isValidationFault(err error) bool {
	return isLocalValidation(err) || businesscentralerrors.IsValidation(err)
}

var documentFaults = []documentFault{
	{
		matches:    isConfigurationFault,
		category:   accountingsync.SyncErrorConfiguration,
		resolution: ProviderName + " is not configured on this server",
	},
	{
		matches:    isAuthFault,
		category:   accountingsync.SyncErrorAuth,
		resolution: "Reconnect " + ProviderName + " from the accounting integration",
	},
	{
		matches:  businesscentralerrors.IsForbidden,
		category: accountingsync.SyncErrorConfiguration,
		resolution: "The signed-in user lacks permission in " + ProviderName +
			". Give them the permission sets the API needs, then retry",
	},
	{matches: isTransientFault, category: accountingsync.SyncErrorTransient},
	{
		matches:  isDuplicateName,
		category: accountingsync.SyncErrorDuplicate,
		resolution: "A record with this name already exists in " + ProviderName +
			". Map the record to it instead, or rename one of them",
	},
	{
		matches:  businesscentralerrors.IsDuplicate,
		category: accountingsync.SyncErrorDuplicate,
		resolution: "The record already exists in " + ProviderName +
			". Skip this record if it is the same one, or change the one in " + ProviderName,
	},
	{
		matches:    businesscentralerrors.IsConflict,
		category:   accountingsync.SyncErrorConflict,
		resolution: "The record changed in " + ProviderName + " while Trenova wrote it. Retry",
	},
	{
		matches:    businesscentralerrors.IsPostingDate,
		category:   accountingsync.SyncErrorClosedPeriod,
		resolution: closedPeriodResolved,
	},
	{
		matches:    businesscentralerrors.IsNotFound,
		category:   accountingsync.SyncErrorNotFound,
		resolution: "The linked record no longer exists in " + ProviderName,
	},
	{
		matches:    isValidationFault,
		category:   accountingsync.SyncErrorValidation,
		resolution: "Correct the record in Trenova or " + ProviderName + ", then retry",
	},
}

func (c *Session) ClassifyDocumentError(err error) *accountingsync.SyncError {
	if err == nil {
		return nil
	}
	var syncErr *accountingsync.SyncError
	if errors.As(err, &syncErr) {
		return syncErr
	}
	out := &accountingsync.SyncError{
		Category:   accountingsync.SyncErrorTransient,
		Code:       errorCode(err),
		Message:    errorMessage(err),
		RetryAfter: businesscentralerrors.RetryAfter(err),
	}
	if businesscentralerrors.IsRateLimited(err) {
		out.Category = accountingsync.SyncErrorRateLimited
		if out.RetryAfter <= 0 {
			out.RetryAfter = defaultRateLimitWait
		}
		return out
	}
	for idx := range documentFaults {
		fault := &documentFaults[idx]
		if fault.matches(err) {
			out.Category = fault.category
			out.Resolution = fault.resolution
			return out
		}
	}
	return out
}

func apiErrorOf(err error) *businesscentralerrors.APIError {
	var apiErr *businesscentralerrors.APIError
	if errors.As(err, &apiErr) {
		return apiErr
	}
	return nil
}

func errorCode(err error) string {
	apiErr := apiErrorOf(err)
	if apiErr == nil {
		return ""
	}
	if apiErr.Code != "" {
		return apiErr.Code
	}
	return strconv.Itoa(apiErr.Status)
}

func errorMessage(err error) string {
	message := err.Error()
	if apiErr := apiErrorOf(err); apiErr != nil && apiErr.Message != "" {
		message = apiErr.Message
	}
	return stringutils.TruncateRunes(message, maxErrorMessage)
}

func isLocalValidation(err error) bool {
	for _, target := range []error{
		businesscentralerrors.ErrIDRequired,
		businesscentralerrors.ErrInvalidID,
		businesscentralerrors.ErrTooManyIDs,
		businesscentralerrors.ErrInvalidCompanyRef,
		businesscentralerrors.ErrInvalidEnvironment,
		businesscentralerrors.ErrInvalidETag,
		businesscentralerrors.ErrInvalidFilter,
		businesscentralerrors.ErrInvalidDate,
		businesscentralerrors.ErrDateRequired,
		businesscentralerrors.ErrUnknownKind,
		businesscentralerrors.ErrUnknownType,
		businesscentralerrors.ErrUnsupportedAction,
		businesscentralerrors.ErrPartyRequired,
		businesscentralerrors.ErrLinesRequired,
		businesscentralerrors.ErrQuantityInvalid,
		businesscentralerrors.ErrPriceInvalid,
		businesscentralerrors.ErrAmountInvalid,
		businesscentralerrors.ErrNameRequired,
		businesscentralerrors.ErrJournalCodeInvalid,
		businesscentralerrors.ErrFieldTooLong,
		businesscentralerrors.ErrFieldNotSupported,
		businesscentralerrors.ErrInvalidText,
		businesscentralerrors.ErrCurrencyCodeInvalid,
		ErrDocumentKind,
		ErrExternalID,
		ErrLinesRequired,
		ErrItemDraftRequired,
		ErrPartyDraftRequired,
		ErrPurchaseDocumentType,
	} {
		if errors.Is(err, target) {
			return true
		}
	}
	return false
}
