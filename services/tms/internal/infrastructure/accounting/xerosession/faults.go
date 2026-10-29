package xerosession

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/xeroerrors"
)

const (
	ProviderName    = "Xero"
	offlineRetry    = 5 * time.Minute
	maxErrorMessage = 2000
)

var (
	ErrDocumentKind = errors.New("xero: this record type is not a document Xero holds")
	ErrInvalidDate  = errors.New("xero: a document date must be written as YYYY-MM-DD")
	ErrExternalID   = errors.New("xero: updating a document needs the id Xero gave it")
	ErrCreditNoteID = errors.New(
		"xero: the credit application does not name the credit note it was allocated from",
	)
)

type documentFault struct {
	matches    func(err error, apiErr *xeroerrors.APIError) bool
	category   accountingsync.SyncErrorCategory
	resolution string
}

func isConfigurationFault(err error, _ *xeroerrors.APIError) bool {
	return errors.Is(err, ErrNotConfigured) || errors.Is(err, ErrNoRedirect)
}

func isAuthFault(err error, _ *xeroerrors.APIError) bool {
	return xeroerrors.IsInvalidGrant(err) || xeroerrors.IsAuth(err) || xeroerrors.IsForbidden(err)
}

func isTransientFault(err error, _ *xeroerrors.APIError) bool {
	return xeroerrors.IsTransient(err) || errors.Is(err, context.DeadlineExceeded)
}

func isDuplicateName(err error, apiErr *xeroerrors.APIError) bool {
	return xeroerrors.IsDuplicateNumber(err) && validationMentions(apiErr, "contact name")
}

func isValidationFault(err error, apiErr *xeroerrors.APIError) bool {
	return isLocalValidation(err) || xeroerrors.IsValidation(err) ||
		hasStatus(apiErr, http.StatusBadRequest)
}

func ignoringAPIError(match func(error) bool) func(error, *xeroerrors.APIError) bool {
	return func(err error, _ *xeroerrors.APIError) bool { return match(err) }
}

func isMappingFaultOf(_ error, apiErr *xeroerrors.APIError) bool {
	return isMappingFault(apiErr)
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
	{matches: isTransientFault, category: accountingsync.SyncErrorTransient},
	{
		matches:  isDuplicateName,
		category: accountingsync.SyncErrorDuplicate,
		resolution: "A contact with this name already exists in " + ProviderName +
			". Map the record to it instead, or rename one of them",
	},
	{
		matches:  ignoringAPIError(xeroerrors.IsDuplicateNumber),
		category: accountingsync.SyncErrorDuplicate,
		resolution: "A document with this number already exists in " + ProviderName +
			". Skip this record if it is the same document, or renumber the one in " +
			ProviderName,
	},
	{
		matches:  ignoringAPIError(xeroerrors.IsLockDate),
		category: accountingsync.SyncErrorClosedPeriod,
		resolution: "The books are locked for this date in " + ProviderName +
			". Move the lock date there or skip this record",
	},
	{
		matches:    ignoringAPIError(xeroerrors.IsNotFound),
		category:   accountingsync.SyncErrorNotFound,
		resolution: "The linked record no longer exists in " + ProviderName,
	},
	{
		matches:  isMappingFaultOf,
		category: accountingsync.SyncErrorMapping,
		resolution: "A mapped " + ProviderName + " record is missing or archived. " +
			"Refresh the reference data and fix the mapping",
	},
	{
		matches:    isValidationFault,
		category:   accountingsync.SyncErrorValidation,
		resolution: "Correct the document in Trenova, then retry",
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

	apiErr := apiErrorOf(err)
	switch {
	case xeroerrors.IsRateLimited(err):
		return rateLimited(err, apiErr)
	case xeroerrors.IsOrganisationOffline(err):
		return &accountingsync.SyncError{
			Category: accountingsync.SyncErrorTransient,
			Code:     "organisation-offline",
			Message: "The " + ProviderName + " organisation is offline. Retry in " +
				offlineRetry.String(),
			RetryAfter: offlineRetry,
		}
	}

	out := &accountingsync.SyncError{
		Category: accountingsync.SyncErrorTransient,
		Code:     errorCode(apiErr),
		Message:  errorMessage(err, apiErr),
	}
	for idx := range documentFaults {
		fault := &documentFaults[idx]
		if fault.matches(err, apiErr) {
			out.Category = fault.category
			out.Resolution = fault.resolution
			return out
		}
	}
	return out
}

func rateLimited(err error, apiErr *xeroerrors.APIError) *accountingsync.SyncError {
	out := &accountingsync.SyncError{
		Category: accountingsync.SyncErrorRateLimited,
		Code:     errorCode(apiErr),
		Message:  errorMessage(err, apiErr),
	}
	if apiErr != nil {
		out.RetryAfter = apiErr.RetryAfter
	}
	if apiErr == nil || apiErr.RateLimitProblem != xeroerrors.RateLimitDay {
		return out
	}
	out.Code = xeroerrors.RateLimitDay
	out.Message = ProviderName + " has refused further calls for this organisation today"
	if apiErr.RetryAfter > 0 {
		out.Message += "; retry after " + apiErr.RetryAfter.String()
	}
	out.Resolution = "Trenova retries when " + ProviderName + "'s daily allowance resets"
	return out
}

func apiErrorOf(err error) *xeroerrors.APIError {
	var apiErr *xeroerrors.APIError
	if errors.As(err, &apiErr) {
		return apiErr
	}
	return nil
}

func hasStatus(apiErr *xeroerrors.APIError, status int) bool {
	return apiErr != nil && apiErr.Status == status
}

func validationMentions(apiErr *xeroerrors.APIError, needles ...string) bool {
	if apiErr == nil {
		return false
	}
	for _, message := range apiErr.ValidationMessages {
		lower := strings.ToLower(message)
		for _, needle := range needles {
			if strings.Contains(lower, needle) {
				return true
			}
		}
	}
	return false
}

func isMappingFault(apiErr *xeroerrors.APIError) bool {
	if !hasStatus(apiErr, http.StatusBadRequest) {
		return false
	}
	return validationMentions(apiErr,
		"account code",
		"is not a valid code",
		"has been archived",
		"is archived",
		"item code",
		"contact could not be found",
	)
}

func errorCode(apiErr *xeroerrors.APIError) string {
	if apiErr == nil {
		return ""
	}
	if apiErr.Type != "" {
		return apiErr.Type
	}
	return strconv.Itoa(apiErr.Status)
}

func errorMessage(err error, apiErr *xeroerrors.APIError) string {
	message := err.Error()
	if apiErr != nil {
		switch {
		case len(apiErr.ValidationMessages) > 0:
			message = strings.Join(apiErr.ValidationMessages, "; ")
		case apiErr.Message != "":
			message = apiErr.Message
		}
	}
	return stringutils.TruncateRunes(message, maxErrorMessage)
}

func isLocalValidation(err error) bool {
	for _, target := range []error{
		xeroerrors.ErrTenantIDRequired,
		xeroerrors.ErrIDRequired,
		xeroerrors.ErrInvalidID,
		xeroerrors.ErrTooManyIDs,
		xeroerrors.ErrIdempotencyKey,
		xeroerrors.ErrDateRequired,
		xeroerrors.ErrContactRequired,
		xeroerrors.ErrLinesRequired,
		xeroerrors.ErrLineDescription,
		xeroerrors.ErrUnknownType,
		xeroerrors.ErrUnknownStatus,
		xeroerrors.ErrUnknownAmountTypes,
		xeroerrors.ErrAmountNotPositive,
		xeroerrors.ErrNegativeRate,
		xeroerrors.ErrTargetRequired,
		xeroerrors.ErrAccountRequired,
		xeroerrors.ErrPaymentsRequired,
		xeroerrors.ErrItemCodeInvalid,
		xeroerrors.ErrItemNameInvalid,
		xeroerrors.ErrContactNameInvalid,
		xeroerrors.ErrFieldTooLong,
		xeroerrors.ErrCurrencyCodeInvalid,
		xeroerrors.ErrEmptyFilter,
		xeroerrors.ErrInvalidFilter,
		ErrDocumentKind,
		ErrInvalidDate,
		ErrExternalID,
		ErrCreditNoteID,
		ErrItemDraftRequired,
		ErrPartyDraftRequired,
	} {
		if errors.Is(err, target) {
			return true
		}
	}
	return false
}
