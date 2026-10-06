package bcconnector

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/shared/businesscentral"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	maxErrorMessage      = 2000
	defaultRateLimitWait = 30 * time.Second
	reverseTransaction   = "Reverse the payment's entries in Business Central " +
		"(Reverse transaction), then mark this record resolved"
)

var errDuplicateName = errors.New(
	"businesscentral: a record with this name already exists in Business Central",
)

func classifyError(err error) accountingsync.ErrorCategory {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, ErrNotConfigured), errors.Is(err, errNoRedirect),
		errors.Is(err, errNoAccess):
		return accountingsync.ErrorCategoryConfiguration
	case businesscentral.IsInvalidGrant(err):
		return accountingsync.ErrorCategoryRevoked
	case businesscentral.IsRateLimited(err):
		return accountingsync.ErrorCategoryRateLimited
	case businesscentral.IsInvalidClient(err), businesscentral.IsAuth(err):
		return accountingsync.ErrorCategoryUnauthorized
	case businesscentral.IsForbidden(err):
		return accountingsync.ErrorCategoryConfiguration
	case businesscentral.IsTransient(err), errors.Is(err, context.DeadlineExceeded):
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
	return errors.Is(err, ErrNotConfigured) || errors.Is(err, errNoRedirect) ||
		errors.Is(err, errNoAccess)
}

func isAuthFault(err error) bool {
	return businesscentral.IsInvalidGrant(err) || businesscentral.IsInvalidClient(err) ||
		businesscentral.IsAuth(err)
}

func isTransientFault(err error) bool {
	return businesscentral.IsTransient(err) || errors.Is(err, context.DeadlineExceeded)
}

func isDuplicateName(err error) bool {
	return errors.Is(err, errDuplicateName)
}

func isValidationFault(err error) bool {
	return isLocalValidation(err) || businesscentral.IsValidation(err)
}

var documentFaults = []documentFault{
	{
		matches:    isConfigurationFault,
		category:   accountingsync.SyncErrorConfiguration,
		resolution: providerName + " is not configured on this server",
	},
	{
		matches:    isAuthFault,
		category:   accountingsync.SyncErrorAuth,
		resolution: "Reconnect " + providerName + " from the accounting integration",
	},
	{
		matches:  businesscentral.IsForbidden,
		category: accountingsync.SyncErrorConfiguration,
		resolution: "The signed-in user lacks permission in " + providerName +
			". Give them the permission sets the API needs, then retry",
	},
	{matches: isTransientFault, category: accountingsync.SyncErrorTransient},
	{
		matches:  isDuplicateName,
		category: accountingsync.SyncErrorDuplicate,
		resolution: "A record with this name already exists in " + providerName +
			". Map the record to it instead, or rename one of them",
	},
	{
		matches:  businesscentral.IsDuplicate,
		category: accountingsync.SyncErrorDuplicate,
		resolution: "The record already exists in " + providerName +
			". Skip this record if it is the same one, or change the one in " + providerName,
	},
	{
		matches:    businesscentral.IsConflict,
		category:   accountingsync.SyncErrorConflict,
		resolution: "The record changed in " + providerName + " while Trenova wrote it. Retry",
	},
	{
		matches:    businesscentral.IsPostingDate,
		category:   accountingsync.SyncErrorClosedPeriod,
		resolution: closedPeriodResolved,
	},
	{
		matches:    businesscentral.IsNotFound,
		category:   accountingsync.SyncErrorNotFound,
		resolution: "The linked record no longer exists in " + providerName,
	},
	{
		matches:    isValidationFault,
		category:   accountingsync.SyncErrorValidation,
		resolution: "Correct the record in Trenova or " + providerName + ", then retry",
	},
}

func (c *Connector) ClassifyDocumentError(err error) *accountingsync.SyncError {
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
		RetryAfter: businesscentral.RetryAfter(err),
	}
	if businesscentral.IsRateLimited(err) {
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

func apiErrorOf(err error) *businesscentral.APIError {
	var apiErr *businesscentral.APIError
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
		businesscentral.ErrIDRequired,
		businesscentral.ErrInvalidID,
		businesscentral.ErrTooManyIDs,
		businesscentral.ErrInvalidCompanyRef,
		businesscentral.ErrInvalidEnvironment,
		businesscentral.ErrInvalidETag,
		businesscentral.ErrInvalidFilter,
		businesscentral.ErrInvalidDate,
		businesscentral.ErrDateRequired,
		businesscentral.ErrUnknownKind,
		businesscentral.ErrUnknownType,
		businesscentral.ErrUnsupportedAction,
		businesscentral.ErrPartyRequired,
		businesscentral.ErrLinesRequired,
		businesscentral.ErrQuantityInvalid,
		businesscentral.ErrPriceInvalid,
		businesscentral.ErrAmountInvalid,
		businesscentral.ErrNameRequired,
		businesscentral.ErrJournalCodeInvalid,
		businesscentral.ErrFieldTooLong,
		businesscentral.ErrFieldNotSupported,
		businesscentral.ErrInvalidText,
		businesscentral.ErrCurrencyCodeInvalid,
		errDocumentKind,
		errExternalID,
		errLinesRequired,
		errItemDraftRequired,
		errPartyDraftRequired,
		errPurchaseDocumentType,
	} {
		if errors.Is(err, target) {
			return true
		}
	}
	return false
}

func unmappedItem(purpose string) *accountingsync.SyncError {
	return &accountingsync.SyncError{
		Category:   accountingsync.SyncErrorMapping,
		Code:       "item",
		Message:    purpose + " has no " + providerName + " item mapped",
		Resolution: "Map it to a " + providerName + " item",
	}
}

func unmappedAccount(purpose string) *accountingsync.SyncError {
	return &accountingsync.SyncError{
		Category:   accountingsync.SyncErrorMapping,
		Code:       "account",
		Message:    purpose + " has no " + providerName + " account mapped",
		Resolution: "Map it to a " + providerName + " account",
	}
}

func negativeLine(purpose string) *accountingsync.SyncError {
	return &accountingsync.SyncError{
		Category: accountingsync.SyncErrorValidation,
		Code:     "negative-line",
		Message: purpose + " is negative, and " + providerName +
			" does not take a negative line on this document",
		Resolution: "Net the line into the others in Trenova, then retry",
	}
}

func paidConflict() *accountingsync.SyncError {
	return &accountingsync.SyncError{
		Category: accountingsync.SyncErrorConflict,
		Code:     "document-paid",
		Message: "The document has a payment or credit applied in " + providerName +
			", which refuses to cancel it",
		Resolution: "Unapply the payment in " + providerName + ", then retry",
	}
}

func reversalRefused() *accountingsync.SyncError {
	return &accountingsync.SyncError{
		Category: accountingsync.SyncErrorConfiguration,
		Code:     "reverse-transaction",
		Message: providerName + "'s API can neither unapply nor reverse a posted payment, " +
			"so Trenova cannot change or void it",
		Resolution: reverseTransaction,
	}
}

func creditApplicationRefused() *accountingsync.SyncError {
	return &accountingsync.SyncError{
		Category: accountingsync.SyncErrorConfiguration,
		Code:     "credit-application",
		Message: providerName + " applies a credit memo only when it is posted against " +
			"an invoice, so Trenova cannot apply or unapply an existing one",
		Resolution: "Apply the credit memo in " + providerName +
			", then mark this record resolved",
	}
}

func unpostedJournal(code string) *accountingsync.SyncError {
	return &accountingsync.SyncError{
		Category: accountingsync.SyncErrorConfiguration,
		Code:     "journal-post",
		Message: providerName + " would not post the " + code +
			" payment journal from Trenova, so the payment waits there unposted",
		Resolution: "Post the " + code + " payment journal in " + providerName,
	}
}

func billLinesUnknown() *accountingsync.SyncError {
	return &accountingsync.SyncError{
		Category: accountingsync.SyncErrorConfiguration,
		Code:     "bill-lines",
		Message: "Trenova does not know which accounts the bill was posted to, " +
			"so it cannot credit it in " + providerName,
		Resolution: "Credit the bill with a purchase credit memo in " + providerName +
			", then mark this record resolved",
	}
}
