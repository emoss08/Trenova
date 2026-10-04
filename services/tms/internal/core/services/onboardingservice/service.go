package onboardingservice

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/core/domain/onboarding"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	organizationPrefix = "organization"
	maxNameLength      = 100
	maxAddressLength   = 150
	maxCityLength      = 100
	maxSCACLength      = 4
	maxDOTNumberLength = 8
	zipPlusFourLength  = 10
	fieldName          = "name"
	fieldTimezone      = "timezone"
	fieldAddressLine1  = "addressLine1"
	fieldCity          = "city"
	fieldStateID       = "stateId"
	fieldPostalCode    = "postalCode"
	fieldOperationType = "operationType"
)

var scacPattern = regexp.MustCompile(`^[A-Z]{2,4}$`)

type OrganizationStore interface {
	GetByID(
		ctx context.Context,
		req repositories.GetOrganizationByIDRequest,
	) (*tenant.Organization, error)
	Update(ctx context.Context, entity *tenant.Organization) (*tenant.Organization, error)
}

type SampleDataLoader interface {
	Load(ctx context.Context, req *SampleDataRequest) error
}

type Params struct {
	fx.In

	DB            ports.DBConnection
	Repository    repositories.OnboardingRepository
	Organizations services.OrganizationService
	Plans         services.PlanService
	SampleData    *SampleData
	Logger        *zap.Logger
}

type Service struct {
	db            ports.DBConnection
	repo          repositories.OnboardingRepository
	organizations OrganizationStore
	plans         services.PlanService
	sampleData    SampleDataLoader
	l             *zap.Logger
}

func New(p Params) services.OnboardingService {
	return &Service{
		db:            p.DB,
		repo:          p.Repository,
		organizations: p.Organizations,
		plans:         p.Plans,
		sampleData:    p.SampleData,
		l:             p.Logger.Named("service.onboarding"),
	}
}

func notRequired() *services.OnboardingState {
	return &services.OnboardingState{
		Required: false,
		Status:   onboarding.StatusCompleted,
	}
}

func (s *Service) Get(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*services.OnboardingState, error) {
	if !s.plans.IsCloud() {
		return notRequired(), nil
	}

	entity, err := s.repo.Get(ctx, repositories.GetOnboardingRequest{TenantInfo: tenantInfo})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return notRequired(), nil
		}
		return nil, err
	}

	org, err := s.organizations.GetByID(ctx, repositories.GetOrganizationByIDRequest{
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}

	return stateOf(entity, org), nil
}

func stateOf(entity *onboarding.Onboarding, org *tenant.Organization) *services.OnboardingState {
	state := &services.OnboardingState{
		Required:         !entity.IsCompleted(),
		Status:           entity.Status,
		OperationType:    entity.OperationType,
		SampleDataLoaded: entity.SampleDataLoaded,
		CompletedAt:      entity.CompletedAt,
	}

	if org != nil {
		state.Organization = organizationOf(org, entity.IsCompleted())
	}

	return state
}

func organizationOf(org *tenant.Organization, completed bool) *services.OnboardingOrganization {
	out := &services.OnboardingOrganization{
		Name:     org.Name,
		Timezone: org.Timezone,
	}
	if !completed {
		return out
	}

	out.AddressLine1 = org.AddressLine1
	out.City = org.City
	out.StateID = org.StateID
	out.PostalCode = org.PostalCode
	if !onboarding.IsPlaceholderSCAC(org.ScacCode) {
		out.ScacCode = org.ScacCode
	}
	if !onboarding.IsPlaceholderDOTNumber(org.DOTNumber) {
		out.DOTNumber = org.DOTNumber
	}

	return out
}

func (s *Service) Complete(
	ctx context.Context,
	req *services.CompleteOnboardingRequest,
) (*services.OnboardingState, error) {
	if !s.plans.IsCloud() {
		return nil, errortypes.NewBusinessError("There is no onboarding to complete")
	}

	normalized, err := normalizeRequest(req)
	if err != nil {
		return nil, err
	}

	var state *services.OnboardingState
	err = s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		entity, txErr := s.repo.Get(txCtx, repositories.GetOnboardingRequest{
			TenantInfo: req.TenantInfo,
		})
		if txErr != nil {
			if errortypes.IsNotFoundError(txErr) {
				return errortypes.NewBusinessError("There is no onboarding to complete")
			}
			return txErr
		}
		if entity.IsCompleted() {
			return errortypes.NewConflictError("Onboarding has already been completed")
		}

		org, txErr := s.updateOrganization(txCtx, req.TenantInfo, normalized)
		if txErr != nil {
			return txErr
		}

		if req.LoadSampleData {
			if txErr = s.sampleData.Load(txCtx, &SampleDataRequest{
				TenantInfo:   req.TenantInfo,
				Actor:        req.Actor,
				Organization: org,
			}); txErr != nil {
				return txErr
			}
		}

		entity.Complete(onboarding.CompleteParams{
			UserID:           req.TenantInfo.UserID,
			OperationType:    normalized.OperationType,
			SampleDataLoaded: req.LoadSampleData,
			CompletedAt:      timeutils.NowUnix(),
		})

		multiErr := errortypes.NewMultiError()
		entity.Validate(multiErr)
		if multiErr.HasErrors() {
			return multiErr
		}

		completed, txErr := s.repo.Complete(txCtx, entity)
		if txErr != nil {
			return txErr
		}

		state = stateOf(completed, org)
		return nil
	})
	if err != nil {
		return nil, err
	}

	s.l.Info("onboarding completed",
		zap.String("organizationId", req.TenantInfo.OrgID.String()),
		zap.String("operationType", normalized.OperationType.String()),
		zap.Bool("sampleData", req.LoadSampleData),
	)

	return state, nil
}

func (s *Service) updateOrganization(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	req *services.CompleteOnboardingRequest,
) (*tenant.Organization, error) {
	org, err := s.organizations.GetByID(ctx, repositories.GetOrganizationByIDRequest{
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}

	profile := req.Organization
	org.Name = profile.Name
	org.Timezone = profile.Timezone
	org.AddressLine1 = profile.AddressLine1
	org.City = profile.City
	org.StateID = profile.StateID
	org.PostalCode = profile.PostalCode
	if profile.ScacCode != "" {
		org.ScacCode = profile.ScacCode
	}
	if profile.DOTNumber != "" {
		org.DOTNumber = profile.DOTNumber
	}
	org.BrokerageEnabled = req.OperationType.RunsBrokerage()
	org.AssetOperationsEnabled = req.OperationType.RunsAssets()

	updated, err := s.organizations.Update(ctx, org)
	if err != nil {
		return nil, prefixOrganizationErrors(err)
	}

	return updated, nil
}

func normalizeRequest(
	req *services.CompleteOnboardingRequest,
) (*services.CompleteOnboardingRequest, error) {
	profile := services.OnboardingOrganization{
		Name:         collapseSpaces(req.Organization.Name),
		Timezone:     strings.TrimSpace(req.Organization.Timezone),
		AddressLine1: collapseSpaces(req.Organization.AddressLine1),
		City:         collapseSpaces(req.Organization.City),
		StateID:      req.Organization.StateID,
		PostalCode:   strings.TrimSpace(req.Organization.PostalCode),
		ScacCode:     strings.ToUpper(strings.TrimSpace(req.Organization.ScacCode)),
		DOTNumber:    strings.TrimSpace(req.Organization.DOTNumber),
	}

	multiErr := errortypes.NewMultiError()
	if !req.OperationType.IsValid() {
		multiErr.Add(
			fieldOperationType,
			errortypes.ErrInvalid,
			"Operation type must be asset, brokerage or both",
		)
	}

	orgErr := multiErr.WithPrefix(organizationPrefix)
	requireText(orgErr, fieldName, profile.Name, maxNameLength, "Enter your company name")
	requireText(orgErr, fieldTimezone, profile.Timezone, maxNameLength, "Select a timezone")
	requireText(orgErr, fieldAddressLine1, profile.AddressLine1, maxAddressLength, "Enter a street address")
	requireText(orgErr, fieldCity, profile.City, maxCityLength, "Enter a city")
	if profile.StateID.IsNil() {
		orgErr.Add(fieldStateID, errortypes.ErrRequired, "Select a state")
	}
	switch {
	case profile.PostalCode == "":
		orgErr.Add(fieldPostalCode, errortypes.ErrRequired, "Enter a ZIP code")
	case domaintypes.ValidatePostalCode(profile.PostalCode) != nil ||
		len(profile.PostalCode) > zipPlusFourLength:
		orgErr.Add(fieldPostalCode, errortypes.ErrInvalidFormat, "Enter a 5-digit ZIP code")
	}
	if profile.ScacCode != "" &&
		(!scacPattern.MatchString(profile.ScacCode) || len(profile.ScacCode) > maxSCACLength) {
		orgErr.Add("scacCode", errortypes.ErrInvalidFormat, "A SCAC is 2 to 4 letters")
	}
	if profile.DOTNumber != "" && !isDigits(profile.DOTNumber, maxDOTNumberLength) {
		orgErr.Add("dotNumber", errortypes.ErrInvalidFormat, "A DOT number is up to 8 digits")
	}

	if multiErr.HasErrors() {
		return nil, multiErr
	}

	return &services.CompleteOnboardingRequest{
		TenantInfo:     req.TenantInfo,
		Actor:          req.Actor,
		Organization:   profile,
		OperationType:  req.OperationType,
		LoadSampleData: req.LoadSampleData,
	}, nil
}

func requireText(multiErr *errortypes.MultiError, field, value string, limit int, message string) {
	switch {
	case value == "":
		multiErr.Add(field, errortypes.ErrRequired, message)
	case utf8.RuneCountInString(value) > limit:
		multiErr.Add(field, errortypes.ErrInvalidLength, "Must be {0} characters or fewer", limit)
	}
}

func prefixOrganizationErrors(err error) error {
	var source *errortypes.MultiError
	if !errors.As(err, &source) || source == nil || !source.HasErrors() {
		return err
	}

	root := errortypes.NewMultiError()
	child := root.WithPrefix(organizationPrefix)
	for _, fieldErr := range source.Errors {
		child.AddError(fieldErr)
	}

	return root
}

func collapseSpaces(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func isDigits(value string, maxLength int) bool {
	if value == "" || len(value) > maxLength {
		return false
	}
	for i := range len(value) {
		if value[i] < '0' || value[i] > '9' {
			return false
		}
	}

	return true
}
