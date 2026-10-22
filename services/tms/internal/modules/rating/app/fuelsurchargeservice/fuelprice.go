package fuelsurchargeservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/rating/domain/fuelsurcharge"
	"github.com/emoss08/trenova/internal/modules/rating/ports/fuelsurchargeports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func (s *Service) FuelPriceAsOf(
	ctx context.Context,
	req *fuelsurchargeports.FuelPriceAsOfRequest,
) (*fuelsurchargeports.FuelPriceAsOf, error) {
	programID, err := s.applicableProgram(ctx, req)
	if err != nil {
		return nil, err
	}
	if programID.IsNotNil() {
		price, priceErr := s.programPriceAsOf(ctx, req, programID)
		if priceErr != nil || price != nil {
			return price, priceErr
		}
	}
	return s.defaultPriceAsOf(ctx, req)
}

func (s *Service) applicableProgram(
	ctx context.Context,
	req *fuelsurchargeports.FuelPriceAsOfRequest,
) (pulid.ID, error) {
	if override := req.Override; override != nil && !override.Waived && override.ProgramID.IsNotNil() {
		return override.ProgramID, nil
	}
	if req.CustomerID.IsNil() {
		return pulid.Nil, nil
	}
	profile, err := s.customerRepo.GetBillingProfile(ctx, customerports.GetCustomerBillingProfileRequest{
		CustomerID: req.CustomerID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return pulid.Nil, nil
		}
		return pulid.Nil, err
	}
	if profile == nil || !profile.AppliesFuelSurcharge() {
		return pulid.Nil, nil
	}
	return *profile.FuelSurchargeProgramID, nil
}

func (s *Service) programPriceAsOf(
	ctx context.Context,
	req *fuelsurchargeports.FuelPriceAsOfRequest,
	programID pulid.ID,
) (*fuelsurchargeports.FuelPriceAsOf, error) {
	program, err := s.programRepo.GetByID(ctx, &fuelsurchargeports.GetFuelSurchargeProgramByIDRequest{
		ProgramID:    programID,
		TenantInfo:   req.TenantInfo,
		IncludeIndex: true,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, nil
		}
		return nil, err
	}
	prices, err := s.priceRepo.GetLatestOnOrBefore(ctx, &fuelsurchargeports.GetLatestFuelPricesRequest{
		FuelIndexID: program.FuelIndexID,
		TenantInfo:  req.TenantInfo,
		Date:        req.Date,
		Limit:       1,
	})
	if err != nil {
		return nil, err
	}
	if len(prices) == 0 || prices[0] == nil {
		return nil, nil
	}
	code := ""
	if program.FuelIndex != nil {
		code = program.FuelIndex.Code
	}
	return &fuelsurchargeports.FuelPriceAsOf{
		Price:     prices[0].Price,
		PriceDate: fuelsurcharge.NormalizePriceDate(prices[0].PriceDate),
		IndexCode: code,
	}, nil
}

func (s *Service) defaultPriceAsOf(
	ctx context.Context,
	req *fuelsurchargeports.FuelPriceAsOfRequest,
) (*fuelsurchargeports.FuelPriceAsOf, error) {
	indices, err := s.indexRepo.ListActive(ctx, req.TenantInfo)
	if err != nil || len(indices) == 0 {
		return nil, err
	}
	pricesByIndex, err := s.priceRepo.LatestPerIndex(ctx, &fuelsurchargeports.LatestPricesPerIndexRequest{
		TenantInfo: req.TenantInfo,
		PerIndex:   1,
		OnOrBefore: req.Date,
	})
	if err != nil {
		return nil, err
	}

	var (
		chosenIndex *fuelsurcharge.FuelIndex
		chosenPrice *fuelsurcharge.FuelIndexPrice
	)
	for _, index := range indices {
		if index == nil {
			continue
		}
		prices := pricesByIndex[index.ID]
		if len(prices) == 0 || prices[0] == nil {
			continue
		}
		candidate := prices[0]
		candidateDate := fuelsurcharge.NormalizePriceDate(candidate.PriceDate)
		if chosenPrice == nil ||
			candidateDate > fuelsurcharge.NormalizePriceDate(chosenPrice.PriceDate) ||
			(candidateDate == fuelsurcharge.NormalizePriceDate(chosenPrice.PriceDate) &&
				index.Code < chosenIndex.Code) {
			chosenIndex, chosenPrice = index, candidate
		}
	}
	if chosenPrice == nil {
		return nil, nil
	}
	return &fuelsurchargeports.FuelPriceAsOf{
		Price:     chosenPrice.Price,
		PriceDate: fuelsurcharge.NormalizePriceDate(chosenPrice.PriceDate),
		IndexCode: chosenIndex.Code,
	}, nil
}

func (s *Service) FuelChargeIDs(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]pulid.ID, error) {
	programs, err := s.programRepo.ListActive(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	ids := make([]pulid.ID, 0, len(programs))
	for _, program := range programs {
		if program != nil && program.AccessorialChargeID.IsNotNil() {
			ids = append(ids, program.AccessorialChargeID)
		}
	}
	return ids, nil
}
