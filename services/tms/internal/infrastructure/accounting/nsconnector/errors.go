package nsconnector

import (
	"context"
	"errors"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/shared/netsuite"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	providerName         = "NetSuite"
	maxErrorMessage      = 2000
	defaultRateLimitWait = 10 * time.Second
	defaultPermission    = "this record"
	closedPeriodResolved = "Reopen the accounting period in NetSuite, or send the record on " +
		"the first open day"
	reversePayment = "Reverse the payment in NetSuite, then mark this record resolved"
	unapplyPayment = "Unapply the payment in NetSuite, then retry"
)

var (
	errDuplicateName = errors.New("netsuite: a record with this name already exists in NetSuite")
	errUnreachable   = errors.New("netsuite: Trenova could not reach this NetSuite account")
	permissionName   = regexp.MustCompile(`'([^']{1,120})'`)
)

func classifyError(err error) accountingsync.ErrorCategory {
	switch {
	case err == nil:
		return ""
	case isConfigurationFault(err):
		return accountingsync.ErrorCategoryConfiguration
	case errors.Is(err, errCertificateRevoked), netsuite.IsInvalidGrant(err):
		return accountingsync.ErrorCategoryRevoked
	case netsuite.IsRateLimited(err):
		return accountingsync.ErrorCategoryRateLimited
	case netsuite.IsInvalidClient(err), netsuite.IsUnauthorized(err):
		return accountingsync.ErrorCategoryUnauthorized
	case netsuite.IsForbidden(err), netsuite.IsPermission(err):
		return accountingsync.ErrorCategoryConfiguration
	case isTransientFault(err):
		return accountingsync.ErrorCategoryTransient
	default:
		return accountingsync.ErrorCategoryUnknown
	}
}

func isConfigurationFault(err error) bool {
	return errors.Is(err, ErrNotConfigured) || errors.Is(err, errNoRedirect) ||
		errors.Is(err, errUnreachable) || errors.Is(err, errNoCompany)
}

func isAuthFault(err error) bool {
	return errors.Is(err, errCertificateRevoked) || netsuite.IsInvalidGrant(err) ||
		netsuite.IsInvalidClient(err) || netsuite.IsUnauthorized(err)
}

func isPermissionFault(err error) bool {
	return netsuite.IsForbidden(err) || netsuite.IsPermission(err)
}

func isQueryRefused(err error) bool {
	if netsuite.IsUnsupportedTable(err) || isPermissionFault(err) {
		return true
	}
	var apiErr *netsuite.APIError
	return errors.As(err, &apiErr) && apiErr.Status == http.StatusBadRequest
}

func isTransientFault(err error) bool {
	return netsuite.IsTransient(err) || errors.Is(err, context.DeadlineExceeded)
}

func isDuplicateFault(err error) bool {
	return errors.Is(err, errDuplicateName) || netsuite.IsIdempotencyConflict(err) ||
		netsuite.IsDuplicateExternalID(err)
}

func isValidationFault(err error) bool {
	if isLocalValidation(err) {
		return true
	}
	code := errorCode(err)
	return code == netsuite.CodeInvalidContent || code == netsuite.CodeInvalidRequest ||
		code == netsuite.CodeUserError
}

func isUnreachable(err error) bool {
	var dnsErr *net.DNSError
	return netsuite.IsNotFound(err) || errors.As(err, &dnsErr)
}

type documentFault struct {
	matches    func(err error) bool
	category   accountingsync.SyncErrorCategory
	resolution string
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
	{matches: isTransientFault, category: accountingsync.SyncErrorTransient},
	{
		matches:  errorIs(errDuplicateName),
		category: accountingsync.SyncErrorDuplicate,
		resolution: "A record with this name already exists in " + providerName +
			". Map the record to it instead, or rename one of them",
	},
	{
		matches:  isDuplicateFault,
		category: accountingsync.SyncErrorDuplicate,
		resolution: "The record already exists in " + providerName +
			". Skip this record if it is the same one, or change the one in " + providerName,
	},
	{
		matches:    netsuite.IsClosedPeriod,
		category:   accountingsync.SyncErrorClosedPeriod,
		resolution: closedPeriodResolved,
	},
	{
		matches:    netsuite.IsNotFound,
		category:   accountingsync.SyncErrorNotFound,
		resolution: "The linked record no longer exists in " + providerName,
	},
	{
		matches:  netsuite.IsInvalidReference,
		category: accountingsync.SyncErrorMapping,
		resolution: "Check the mapping for the record " + providerName +
			" named, then retry",
	},
	{
		matches:    isValidationFault,
		category:   accountingsync.SyncErrorValidation,
		resolution: "Correct the record in Trenova or " + providerName + ", then retry",
	},
}

func errorIs(target error) func(error) bool {
	return func(err error) bool { return errors.Is(err, target) }
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
		RetryAfter: netsuite.RetryAfter(err),
	}
	switch {
	case netsuite.IsRateLimited(err):
		out.Category = accountingsync.SyncErrorRateLimited
		if out.RetryAfter <= 0 {
			out.RetryAfter = defaultRateLimitWait
		}
		return out
	case isPermissionFault(err) && !isAuthFault(err):
		record := permissionRecord(err)
		out.Category = accountingsync.SyncErrorConfiguration
		out.Message = "The role lacks permission for " + record + " in " + providerName
		out.Resolution = "Give the " + providerName + " role the permission for " + record +
			", then retry"
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

func apiErrorOf(err error) *netsuite.APIError {
	var apiErr *netsuite.APIError
	if errors.As(err, &apiErr) {
		return apiErr
	}
	return nil
}

func errorCode(err error) string {
	apiErr := apiErrorOf(err)
	if apiErr == nil {
		var oauthErr *netsuite.OAuthError
		if errors.As(err, &oauthErr) {
			return oauthErr.Code
		}
		return ""
	}
	if code := apiErr.Code(); code != "" {
		return code
	}
	return strconv.Itoa(apiErr.Status)
}

func errorMessage(err error) string {
	message := err.Error()
	if apiErr := apiErrorOf(err); apiErr != nil {
		if detail := apiErr.Detail(); detail != "" {
			message = detail
		} else if apiErr.Title != "" {
			message = apiErr.Title
		}
	}
	return stringutils.TruncateRunes(message, maxErrorMessage)
}

func permissionRecord(err error) string {
	apiErr := apiErrorOf(err)
	if apiErr == nil {
		return defaultPermission
	}
	for _, text := range []string{apiErr.Detail(), apiErr.Title} {
		if match := permissionName.FindStringSubmatch(text); len(match) == 2 {
			return strings.TrimSpace(match[1])
		}
	}
	return defaultPermission
}

func deleteRefused(err error) bool {
	if netsuite.IsClosedPeriod(err) {
		return true
	}
	return errorCode(err) == netsuite.CodeUserError && !netsuite.IsPermission(err)
}

func isLocalValidation(err error) bool {
	for _, target := range []error{
		netsuite.ErrInvalidAccountID,
		netsuite.ErrInvalidSubsidiaryID,
		netsuite.ErrInvalidCompanyRef,
		netsuite.ErrInvalidRecordType,
		netsuite.ErrInvalidRecordID,
		netsuite.ErrInvalidExternalID,
		netsuite.ErrInvalidFieldName,
		netsuite.ErrBodyRequired,
		netsuite.ErrInvalidQueryTime,
		errDocumentKind,
		errExternalID,
		errRequestID,
		errLinesRequired,
		errAmountInvalid,
		errApplyTarget,
		errItemDraftRequired,
		errPartyDraftRequired,
		errNameRequired,
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
		Code:     "document-applied",
		Message: "A payment or credit is still applied to the document in " + providerName +
			", so Trenova cannot delete or reverse it",
		Resolution: unapplyPayment,
	}
}

func reversalRefused() *accountingsync.SyncError {
	return &accountingsync.SyncError{
		Category: accountingsync.SyncErrorConfiguration,
		Code:     "reverse-payment",
		Message: "The payment is in a closed or locked period in " + providerName +
			", which has no REST void, so Trenova cannot delete it",
		Resolution: reversePayment,
	}
}

func creditReversalRefused(label string) *accountingsync.SyncError {
	return &accountingsync.SyncError{
		Category: accountingsync.SyncErrorConfiguration,
		Code:     "reverse-credit",
		Message: "The " + label + " is in a closed or locked period in " + providerName +
			", which refuses to delete it",
		Resolution: "Reverse the " + label + " in " + providerName +
			", then mark this record resolved",
	}
}

func creditApplicationUnknown() *accountingsync.SyncError {
	return &accountingsync.SyncError{
		Category: accountingsync.SyncErrorConfiguration,
		Code:     "credit-application",
		Message: "Trenova does not know which credit memo and invoice the application " +
			"joined in " + providerName,
		Resolution: "Unapply the credit memo in " + providerName +
			", then mark this record resolved",
	}
}
