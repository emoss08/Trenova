package nsbinding

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/shared/netsuiteerrors"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	ProviderName         = "NetSuite"
	maxErrorMessage      = 2000
	defaultRateLimitWait = 10 * time.Second
	defaultPermission    = "this record"
	ClosedPeriodResolved = "Reopen the accounting period in NetSuite, or send the record on " +
		"the first open day"
)

var (
	ErrDuplicateName = errors.New("netsuite: a record with this name already exists in NetSuite")
	ErrUnreachable   = errors.New("netsuite: Trenova could not reach this NetSuite account")
	permissionName   = regexp.MustCompile(`'([^']{1,120})'`)
)

func ClassifyError(err error) accountingsync.ErrorCategory {
	switch {
	case err == nil:
		return ""
	case isConfigurationFault(err):
		return accountingsync.ErrorCategoryConfiguration
	case errors.Is(err, ErrCertificateRevoked), netsuiteerrors.IsInvalidGrant(err):
		return accountingsync.ErrorCategoryRevoked
	case netsuiteerrors.IsRateLimited(err):
		return accountingsync.ErrorCategoryRateLimited
	case netsuiteerrors.IsInvalidClient(err), netsuiteerrors.IsUnauthorized(err):
		return accountingsync.ErrorCategoryUnauthorized
	case netsuiteerrors.IsForbidden(err), netsuiteerrors.IsPermission(err):
		return accountingsync.ErrorCategoryConfiguration
	case isTransientFault(err):
		return accountingsync.ErrorCategoryTransient
	default:
		return accountingsync.ErrorCategoryUnknown
	}
}

func isConfigurationFault(err error) bool {
	return errors.Is(err, ErrNotConfigured) || errors.Is(err, ErrNoRedirect) ||
		errors.Is(err, ErrUnreachable) || errors.Is(err, ErrNoCompany)
}

func isAuthFault(err error) bool {
	return errors.Is(err, ErrCertificateRevoked) || netsuiteerrors.IsInvalidGrant(err) ||
		netsuiteerrors.IsInvalidClient(err) || netsuiteerrors.IsUnauthorized(err)
}

func IsPermissionFault(err error) bool {
	return netsuiteerrors.IsForbidden(err) || netsuiteerrors.IsPermission(err)
}

func isQueryRefused(err error) bool {
	if netsuiteerrors.IsUnsupportedTable(err) || IsPermissionFault(err) {
		return true
	}
	var apiErr *netsuiteerrors.APIError
	return errors.As(err, &apiErr) && apiErr.Status == http.StatusBadRequest
}

func isTransientFault(err error) bool {
	return netsuiteerrors.IsTransient(err) || errors.Is(err, context.DeadlineExceeded)
}

func isDuplicateFault(err error) bool {
	return errors.Is(err, ErrDuplicateName) || netsuiteerrors.IsIdempotencyConflict(err) ||
		netsuiteerrors.IsDuplicateExternalID(err)
}

func isValidationFault(err error) bool {
	if isLocalValidation(err) {
		return true
	}
	code := ErrorCode(err)
	return code == netsuiteerrors.CodeInvalidContent || code == netsuiteerrors.CodeInvalidRequest ||
		code == netsuiteerrors.CodeUserError
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
		resolution: ProviderName + " is not configured on this server",
	},
	{
		matches:    isAuthFault,
		category:   accountingsync.SyncErrorAuth,
		resolution: "Reconnect " + ProviderName + " from the accounting integration",
	},
	{matches: isTransientFault, category: accountingsync.SyncErrorTransient},
	{
		matches:  errorIs(ErrDuplicateName),
		category: accountingsync.SyncErrorDuplicate,
		resolution: "A record with this name already exists in " + ProviderName +
			". Map the record to it instead, or rename one of them",
	},
	{
		matches:  isDuplicateFault,
		category: accountingsync.SyncErrorDuplicate,
		resolution: "The record already exists in " + ProviderName +
			". Skip this record if it is the same one, or change the one in " + ProviderName,
	},
	{
		matches:    netsuiteerrors.IsClosedPeriod,
		category:   accountingsync.SyncErrorClosedPeriod,
		resolution: ClosedPeriodResolved,
	},
	{
		matches:    netsuiteerrors.IsNotFound,
		category:   accountingsync.SyncErrorNotFound,
		resolution: "The linked record no longer exists in " + ProviderName,
	},
	{
		matches:  netsuiteerrors.IsInvalidReference,
		category: accountingsync.SyncErrorMapping,
		resolution: "Check the mapping for the record " + ProviderName +
			" named, then retry",
	},
	{
		matches:    isValidationFault,
		category:   accountingsync.SyncErrorValidation,
		resolution: "Correct the record in Trenova or " + ProviderName + ", then retry",
	},
}

func errorIs(target error) func(error) bool {
	return func(err error) bool { return errors.Is(err, target) }
}

func (c *Binding) ClassifyDocumentError(err error) *accountingsync.SyncError {
	if err == nil {
		return nil
	}
	var syncErr *accountingsync.SyncError
	if errors.As(err, &syncErr) {
		return syncErr
	}
	out := &accountingsync.SyncError{
		Category:   accountingsync.SyncErrorTransient,
		Code:       ErrorCode(err),
		Message:    errorMessage(err),
		RetryAfter: netsuiteerrors.RetryAfter(err),
	}
	switch {
	case netsuiteerrors.IsRateLimited(err):
		out.Category = accountingsync.SyncErrorRateLimited
		if out.RetryAfter <= 0 {
			out.RetryAfter = defaultRateLimitWait
		}
		return out
	case IsPermissionFault(err) && !isAuthFault(err):
		record := permissionRecord(err)
		out.Category = accountingsync.SyncErrorConfiguration
		out.Message = "The role lacks permission for " + record + " in " + ProviderName
		out.Resolution = "Give the " + ProviderName + " role the permission for " + record +
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

func apiErrorOf(err error) *netsuiteerrors.APIError {
	var apiErr *netsuiteerrors.APIError
	if errors.As(err, &apiErr) {
		return apiErr
	}
	return nil
}

func ErrorCode(err error) string {
	apiErr := apiErrorOf(err)
	if apiErr == nil {
		var oauthErr *netsuiteerrors.OAuthError
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

func isLocalValidation(err error) bool {
	for _, target := range []error{
		netsuiteerrors.ErrInvalidAccountID,
		netsuiteerrors.ErrInvalidSubsidiaryID,
		netsuiteerrors.ErrInvalidCompanyRef,
		netsuiteerrors.ErrInvalidRecordType,
		netsuiteerrors.ErrInvalidRecordID,
		netsuiteerrors.ErrInvalidExternalID,
		netsuiteerrors.ErrInvalidFieldName,
		netsuiteerrors.ErrBodyRequired,
		netsuiteerrors.ErrInvalidQueryTime,
		ErrDocumentKind,
		ErrExternalID,
		ErrRequestID,
		ErrLinesRequired,
		ErrAmountInvalid,
		ErrApplyTarget,
		ErrItemDraftRequired,
		ErrPartyDraftRequired,
		ErrNameRequired,
		ErrPurchaseDocumentType,
	} {
		if errors.Is(err, target) {
			return true
		}
	}
	return false
}
