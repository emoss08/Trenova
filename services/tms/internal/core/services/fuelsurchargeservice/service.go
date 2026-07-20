package fuelsurchargeservice

import (
	"context"
	"net/http"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/fuelsurcharge"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/internal/core/services/integrationservice"
	"github.com/emoss08/trenova/internal/core/services/notificationservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger              *zap.Logger
	IndexRepo           repositories.FuelIndexRepository
	PriceRepo           repositories.FuelIndexPriceRepository
	ProgramRepo         repositories.FuelSurchargeProgramRepository
	CustomerRepo        repositories.CustomerRepository
	OrgCacheRepo        repositories.OrganizationCacheRepository
	IntegrationService  *integrationservice.Service
	NotificationService *notificationservice.Service
	AuditService        services.AuditService
	HTTPClient          *http.Client `optional:"true"`
}

type Service struct {
	l                   *zap.Logger
	indexRepo           repositories.FuelIndexRepository
	priceRepo           repositories.FuelIndexPriceRepository
	programRepo         repositories.FuelSurchargeProgramRepository
	customerRepo        repositories.CustomerRepository
	orgCacheRepo        repositories.OrganizationCacheRepository
	integrationService  *integrationservice.Service
	notificationService *notificationservice.Service
	auditService        services.AuditService
	httpClient          *http.Client
	now                 func() int64
}

func New(p Params) *Service {
	httpClient := p.HTTPClient
	if httpClient == nil {
		httpClient = newDefaultHTTPClient()
	}

	return &Service{
		l:                   p.Logger.Named("service.fuel-surcharge"),
		indexRepo:           p.IndexRepo,
		priceRepo:           p.PriceRepo,
		programRepo:         p.ProgramRepo,
		customerRepo:        p.CustomerRepo,
		orgCacheRepo:        p.OrgCacheRepo,
		integrationService:  p.IntegrationService,
		notificationService: p.NotificationService,
		auditService:        p.AuditService,
		httpClient:          httpClient,
		now:                 timeutils.NowUnix,
	}
}

func newDefaultHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 15 * time.Second,
		Transport: &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   10,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 10 * time.Second,
			ExpectContinueTimeout: time.Second,
		},
	}
}

func (s *Service) CreateIndex(
	ctx context.Context,
	entity *fuelsurcharge.FuelIndex,
	userID pulid.ID,
) (*fuelsurcharge.FuelIndex, error) {
	if multiErr := validateIndex(entity); multiErr != nil {
		return nil, multiErr
	}

	created, err := s.indexRepo.Create(ctx, entity)
	if err != nil {
		s.l.Error("failed to create fuel index", zap.Error(err))
		return nil, err
	}

	s.logAudit(auditParams{
		resourceID: created.ID.String(),
		operation:  permission.OpCreate,
		userID:     userID,
		current:    created,
		orgID:      created.OrganizationID,
		buID:       created.BusinessUnitID,
		comment:    "Fuel index created",
	})

	return created, nil
}

func (s *Service) UpdateIndex(
	ctx context.Context,
	entity *fuelsurcharge.FuelIndex,
	userID pulid.ID,
) (*fuelsurcharge.FuelIndex, error) {
	if multiErr := validateIndex(entity); multiErr != nil {
		return nil, multiErr
	}

	original, err := s.indexRepo.GetByID(ctx, &repositories.GetFuelIndexByIDRequest{
		FuelIndexID: entity.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
	})
	if err != nil {
		return nil, err
	}

	if original.Source == fuelsurcharge.IndexSourceEIA &&
		entity.Source != fuelsurcharge.IndexSourceEIA {
		multiErr := errortypes.NewMultiError()
		multiErr.Add("source", errortypes.ErrInvalid,
			"EIA indices cannot be converted to custom indices")
		return nil, multiErr
	}

	updated, err := s.indexRepo.Update(ctx, entity)
	if err != nil {
		s.l.Error("failed to update fuel index", zap.Error(err))
		return nil, err
	}

	s.logAudit(auditParams{
		resourceID: updated.ID.String(),
		operation:  permission.OpUpdate,
		userID:     userID,
		current:    updated,
		previous:   original,
		orgID:      updated.OrganizationID,
		buID:       updated.BusinessUnitID,
		comment:    "Fuel index updated",
	})

	return updated, nil
}

func (s *Service) DeleteIndex(
	ctx context.Context,
	req *repositories.GetFuelIndexByIDRequest,
	userID pulid.ID,
) error {
	existing, err := s.indexRepo.GetByID(ctx, req)
	if err != nil {
		return err
	}

	if err = s.indexRepo.Delete(ctx, req); err != nil {
		s.l.Error("failed to delete fuel index", zap.Error(err))
		return err
	}

	s.logAudit(auditParams{
		resourceID: existing.ID.String(),
		operation:  permission.OpDelete,
		userID:     userID,
		previous:   existing,
		orgID:      existing.OrganizationID,
		buID:       existing.BusinessUnitID,
		comment:    "Fuel index deleted",
	})

	return nil
}

func (s *Service) AddManualPrice(
	ctx context.Context,
	entity *fuelsurcharge.FuelIndexPrice,
	userID pulid.ID,
) (*fuelsurcharge.FuelIndexPrice, error) {
	index, err := s.indexRepo.GetByID(ctx, &repositories.GetFuelIndexByIDRequest{
		FuelIndexID: entity.FuelIndexID,
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
	})
	if err != nil {
		return nil, err
	}

	if index.Source != fuelsurcharge.IndexSourceCustom {
		multiErr := errortypes.NewMultiError()
		multiErr.Add("fuelIndexId", errortypes.ErrInvalid,
			"Manual prices can only be added to custom indices")
		return nil, multiErr
	}

	entity.IsManual = true
	entity.EnteredByID = &userID
	entity.Currency = index.Currency

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	created, err := s.priceRepo.Create(ctx, entity)
	if err != nil {
		s.l.Error("failed to create manual fuel price", zap.Error(err))
		return nil, err
	}

	return created, nil
}

func (s *Service) UpdateManualPrice(
	ctx context.Context,
	entity *fuelsurcharge.FuelIndexPrice,
	userID pulid.ID,
) (*fuelsurcharge.FuelIndexPrice, error) {
	tenantInfo := pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	}

	existing, err := s.priceRepo.GetByID(ctx, &repositories.GetFuelIndexPriceByIDRequest{
		PriceID:    entity.ID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}

	if !existing.IsManual {
		multiErr := errortypes.NewMultiError()
		multiErr.Add("id", errortypes.ErrInvalid,
			"Automatically ingested prices cannot be edited")
		return nil, multiErr
	}

	entity.FuelIndexID = existing.FuelIndexID
	entity.IsManual = true
	entity.EnteredByID = &userID
	entity.Currency = existing.Currency

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	updated, err := s.priceRepo.Update(ctx, entity)
	if err != nil {
		s.l.Error("failed to update manual fuel price", zap.Error(err))
		return nil, err
	}

	return updated, nil
}

func (s *Service) DeleteManualPrice(
	ctx context.Context,
	req *repositories.GetFuelIndexPriceByIDRequest,
) error {
	existing, err := s.priceRepo.GetByID(ctx, req)
	if err != nil {
		return err
	}

	if !existing.IsManual {
		multiErr := errortypes.NewMultiError()
		multiErr.Add("id", errortypes.ErrInvalid,
			"Automatically ingested prices cannot be deleted")
		return multiErr
	}

	return s.priceRepo.Delete(ctx, req)
}

func (s *Service) CreateProgram(
	ctx context.Context,
	entity *fuelsurcharge.FuelSurchargeProgram,
	userID pulid.ID,
) (*fuelsurcharge.FuelSurchargeProgram, error) {
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	created, err := s.programRepo.Create(ctx, entity)
	if err != nil {
		s.l.Error("failed to create fuel surcharge program", zap.Error(err))
		return nil, err
	}

	s.logAudit(auditParams{
		resourceID: created.ID.String(),
		operation:  permission.OpCreate,
		userID:     userID,
		current:    created,
		orgID:      created.OrganizationID,
		buID:       created.BusinessUnitID,
		comment:    "Fuel surcharge program created",
	})

	return created, nil
}

func (s *Service) UpdateProgram(
	ctx context.Context,
	entity *fuelsurcharge.FuelSurchargeProgram,
	userID pulid.ID,
) (*fuelsurcharge.FuelSurchargeProgram, error) {
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	original, err := s.programRepo.GetByID(ctx, &repositories.GetFuelSurchargeProgramByIDRequest{
		ProgramID: entity.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
		IncludeRows: true,
	})
	if err != nil {
		return nil, err
	}

	updated, err := s.programRepo.Update(ctx, entity)
	if err != nil {
		s.l.Error("failed to update fuel surcharge program", zap.Error(err))
		return nil, err
	}

	s.logAudit(auditParams{
		resourceID: updated.ID.String(),
		operation:  permission.OpUpdate,
		userID:     userID,
		current:    updated,
		previous:   original,
		orgID:      updated.OrganizationID,
		buID:       updated.BusinessUnitID,
		comment:    "Fuel surcharge program updated",
	})

	return updated, nil
}

func (s *Service) DeleteProgram(
	ctx context.Context,
	req *repositories.GetFuelSurchargeProgramByIDRequest,
	userID pulid.ID,
) error {
	existing, err := s.programRepo.GetByID(ctx, req)
	if err != nil {
		return err
	}

	if err = s.programRepo.Delete(ctx, req); err != nil {
		s.l.Error("failed to delete fuel surcharge program", zap.Error(err))
		return err
	}

	s.logAudit(auditParams{
		resourceID: existing.ID.String(),
		operation:  permission.OpDelete,
		userID:     userID,
		previous:   existing,
		orgID:      existing.OrganizationID,
		buID:       existing.BusinessUnitID,
		comment:    "Fuel surcharge program deleted",
	})

	return nil
}

type IndexLatestPrice struct {
	Index    *fuelsurcharge.FuelIndex
	Latest   *fuelsurcharge.FuelIndexPrice
	Previous *fuelsurcharge.FuelIndexPrice
}

func (s *Service) Dashboard(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*IndexLatestPrice, error) {
	indices, err := s.indexRepo.ListActive(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	pricesByIndex, err := s.priceRepo.LatestPerIndex(ctx, &repositories.LatestPricesPerIndexRequest{
		TenantInfo: tenantInfo,
		PerIndex:   2,
	})
	if err != nil {
		return nil, err
	}

	result := make([]*IndexLatestPrice, 0, len(indices))
	for _, index := range indices {
		entry := &IndexLatestPrice{Index: index}
		prices := pricesByIndex[index.ID]
		if len(prices) > 0 {
			entry.Latest = prices[0]
		}
		if len(prices) > 1 {
			entry.Previous = prices[1]
		}
		result = append(result, entry)
	}

	return result, nil
}

type ProgramCurrentRate struct {
	Program      *fuelsurcharge.FuelSurchargeProgram
	Price        *fuelsurcharge.FuelIndexPrice
	RatePerMile  *decimal.Decimal
	Percent      *decimal.Decimal
	FlatAmount   *decimal.Decimal
	MatchedRow   *fuelsurcharge.FuelSurchargeTableRow
	UsedFallback bool
}

func (s *Service) ProgramCurrentRates(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*ProgramCurrentRate, error) {
	programs, err := s.programRepo.ListActive(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	today := time.Unix(s.now(), 0).UTC()
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)

	result := make([]*ProgramCurrentRate, 0, len(programs))
	for _, program := range programs {
		entry := &ProgramCurrentRate{Program: program}

		prices, pErr := s.priceRepo.GetLatestOnOrBefore(
			ctx,
			&repositories.GetLatestFuelPricesRequest{
				FuelIndexID: program.FuelIndexID,
				TenantInfo:  tenantInfo,
				Date:        today.Format(fuelsurcharge.PriceDateLayout),
				Limit:       3,
			},
		)
		if pErr != nil {
			return nil, pErr
		}

		price, usedFallback, ok := SelectPrice(prices, program, today)
		if ok {
			entry.Price = price
			entry.UsedFallback = usedFallback
			s.populateCurrentRate(entry, program, price.Price)
		}

		result = append(result, entry)
	}

	return result, nil
}

func (s *Service) populateCurrentRate(
	entry *ProgramCurrentRate,
	program *fuelsurcharge.FuelSurchargeProgram,
	price decimal.Decimal,
) {
	switch program.Method {
	case fuelsurcharge.ProgramMethodPerMileStep,
		fuelsurcharge.ProgramMethodPerMileMPG,
		fuelsurcharge.ProgramMethodTablePerMile:
		rate, err := ComputeRatePerMile(program, price)
		if err != nil {
			return
		}
		entry.RatePerMile = &rate
		if program.Method == fuelsurcharge.ProgramMethodTablePerMile {
			entry.MatchedRow = MatchTableRow(program.TableRows, price)
		}
	case fuelsurcharge.ProgramMethodTablePercent:
		row := MatchTableRow(program.TableRows, price)
		if row == nil {
			return
		}
		entry.MatchedRow = row
		percent := row.Value
		entry.Percent = &percent
	case fuelsurcharge.ProgramMethodTableFlat:
		row := MatchTableRow(program.TableRows, price)
		if row == nil {
			return
		}
		entry.MatchedRow = row
		flat := row.Value
		entry.FlatAmount = &flat
	}
}

func validateIndex(entity *fuelsurcharge.FuelIndex) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

type auditParams struct {
	resourceID string
	operation  permission.Operation
	userID     pulid.ID
	current    any
	previous   any
	orgID      pulid.ID
	buID       pulid.ID
	comment    string
}

func (s *Service) logAudit(p auditParams) {
	params := &services.LogActionParams{
		Resource:       permission.ResourceFuelSurchargeProgram,
		ResourceID:     p.resourceID,
		Operation:      p.operation,
		UserID:         p.userID,
		OrganizationID: p.orgID,
		BusinessUnitID: p.buID,
	}

	if p.current != nil {
		params.CurrentState = jsonutils.MustToJSON(p.current)
	}
	if p.previous != nil {
		params.PreviousState = jsonutils.MustToJSON(p.previous)
	}

	if err := s.auditService.LogAction(params,
		auditservice.WithComment(p.comment),
	); err != nil {
		s.l.Error("failed to log audit action", zap.Error(err))
	}
}
