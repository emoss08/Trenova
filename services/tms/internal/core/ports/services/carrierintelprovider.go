package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/carrierintel"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/restx"
)

type CarrierIntelErrorKind string

const (
	CarrierIntelErrorUnauthorized    = CarrierIntelErrorKind("Unauthorized")
	CarrierIntelErrorPaymentRequired = CarrierIntelErrorKind("PaymentRequired")
	CarrierIntelErrorRateLimited     = CarrierIntelErrorKind("RateLimited")
	CarrierIntelErrorNotFound        = CarrierIntelErrorKind("NotFound")
	CarrierIntelErrorInvalidRequest  = CarrierIntelErrorKind("InvalidRequest")
	CarrierIntelErrorUnavailable     = CarrierIntelErrorKind("Unavailable")
	CarrierIntelErrorUnsupported     = CarrierIntelErrorKind("Unsupported")
)

type CarrierIntelProviderError struct {
	Provider   integration.Type
	Kind       CarrierIntelErrorKind
	StatusCode int
	RetryAfter time.Duration
	Message    string
	Err        error
}

func (e *CarrierIntelProviderError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("%s %s: %s", e.Provider, e.Kind, e.Message)
	}
	return fmt.Sprintf("%s %s", e.Provider, e.Kind)
}

func (e *CarrierIntelProviderError) Unwrap() error { return e.Err }

func CarrierIntelErrorKindOf(err error) (CarrierIntelErrorKind, bool) {
	var providerErr *CarrierIntelProviderError
	if errors.As(err, &providerErr) {
		return providerErr.Kind, true
	}
	return "", false
}

func IsCarrierIntelErrorKind(err error, kind CarrierIntelErrorKind) bool {
	got, ok := CarrierIntelErrorKindOf(err)
	return ok && got == kind
}

type CarrierIntelIdentifier struct {
	DOTNumber    string
	DocketNumber string
	Company      string
	EIN          string
	Email        string
	Phone        string
}

func (i CarrierIntelIdentifier) IsZero() bool {
	return i.DOTNumber == "" && i.DocketNumber == "" && i.Company == "" && i.EIN == "" &&
		i.Email == "" && i.Phone == ""
}

type CarrierIntelLookupRequest struct {
	Identifier CarrierIntelIdentifier
	Depth      carrierintel.LookupDepth
}

type CarrierIntelLookupResult struct {
	Profile     *carrierintel.Profile
	Raw         []byte
	Endpoint    carrierintel.Endpoint
	ProviderRef string
	Depth       carrierintel.LookupDepth
	NotFound    bool
	SourceAsOf  *int64
}

type CarrierIntelCall struct {
	Endpoint   carrierintel.Endpoint
	DOTNumber  string
	StatusCode int
	Latency    time.Duration
	Found      bool
	Units      int
	Err        error
}

type CarrierIntelCallRecorder func(ctx context.Context, call CarrierIntelCall)

type CarrierIntelBindParams struct {
	TenantInfo pagination.TenantInfo
	Config     map[string]string
	Limiter    restx.Limiter
	Recorder   CarrierIntelCallRecorder
}

type CarrierIntelConnector interface {
	IntegrationType() integration.Type
	Descriptor() carrierintel.ProviderDescriptor
	PriceBook() carrierintel.PriceBook
	TestConnection(ctx context.Context, config map[string]string) error
	Bind(params *CarrierIntelBindParams) (CarrierIntelClient, error)
}

type CarrierIntelClient interface {
	Provider() integration.Type
	IsSandbox() bool
	Lookup(ctx context.Context, req *CarrierIntelLookupRequest) (*CarrierIntelLookupResult, error)
}

type CarrierIntelSearchRequest struct {
	Query       string
	CompanyName string
	EIN         string
	VIN         string
	State       string
	Limit       int
	Offset      int
}

type CarrierIntelSearchHit struct {
	Profile     *carrierintel.Profile
	ProviderRef string
}

type CarrierIntelSearchResult struct {
	Items []CarrierIntelSearchHit
	Total int
}

type CarrierIntelSuggestion struct {
	DOTNumber string
	LegalName string
	DBAName   string
	City      string
	State     string
}

type CarrierIntelSearcher interface {
	Search(ctx context.Context, req *CarrierIntelSearchRequest) (*CarrierIntelSearchResult, error)
	Autocomplete(ctx context.Context, query string, limit int) ([]CarrierIntelSuggestion, error)
}

type CarrierIntelEnrollFailure struct {
	ProviderRef string
	Message     string
}

type CarrierIntelEnrollResult struct {
	Succeeded []string
	Failed    []CarrierIntelEnrollFailure
}

type CarrierIntelVendorChange struct {
	VendorField string
	Path        string
	Section     carrierintel.Section
	Prior       any
	Current     any
}

type CarrierIntelChangedProfile struct {
	ProviderRef string
	DOTNumber   string
	ChangedAt   *int64
	Changes     []CarrierIntelVendorChange
}

type CarrierIntelChangeFeedRequest struct {
	Since    int64
	Until    int64
	Page     int
	PageSize int
}

type CarrierIntelChangeFeedPage struct {
	Items    []CarrierIntelChangedProfile
	Total    int
	Page     int
	HasMore  bool
	PageSize int
}

type CarrierIntelWatchlistPage struct {
	ProviderRefs []string
	Total        int
	HasMore      bool
}

type CarrierIntelNativeMonitor interface {
	MonitoringRef(profile *carrierintel.Profile, dotNumber, docketNumber string) string
	Enroll(ctx context.Context, refs []string) (*CarrierIntelEnrollResult, error)
	Unenroll(ctx context.Context, refs []string) (*CarrierIntelEnrollResult, error)
	ListChanges(
		ctx context.Context,
		req *CarrierIntelChangeFeedRequest,
	) (*CarrierIntelChangeFeedPage, error)
	ListWatchlist(ctx context.Context, page, pageSize int) (*CarrierIntelWatchlistPage, error)
}

type CarrierIntelEquipmentRequest struct {
	VIN         string
	PlateNumber string
	PlateState  string
	UnitNumber  string
	UnitType    carrierintel.UnitType
}

type CarrierIntelEquipmentMatch struct {
	DOTNumber string
	LegalName string
	Unit      *carrierintel.Equipment
	Profile   *carrierintel.Profile
}

type CarrierIntelEquipmentResult struct {
	Matches []CarrierIntelEquipmentMatch
	Raw     []byte
}

type CarrierIntelEquipmentLookup interface {
	FindByEquipment(
		ctx context.Context,
		req *CarrierIntelEquipmentRequest,
	) (*CarrierIntelEquipmentResult, error)
}
