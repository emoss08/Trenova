package selectoptionsresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/accountingselectoptions"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/agentselectoptions"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/documentsselectoptions"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/ediselectoptions"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/equipmentselectoptions"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/identityselectoptions"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/operationsselectoptions"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/partiesselectoptions"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/placesselectoptions"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/ratingselectoptions"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/selectoption"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/settlementselectoptions"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/workforceselectoptions"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

type SelectOptionRegistryEntry struct {
	Resolve  func(context.Context, *selectoption.Request) (*gqlmodel.SelectOptionConnection, error)
	parseIDs func([]string) ([]pulid.ID, error)
}

func (e SelectOptionRegistryEntry) IDs(values []string) ([]pulid.ID, error) {
	if e.parseIDs != nil {
		return e.parseIDs(values)
	}
	return gqlid.ParseIDs(values)
}

func (r *Deps) RequireAuthContext(ctx context.Context) (*authctx.AuthContext, error) {
	authCtx, ok := gqlctx.AuthContext(ctx)
	if !ok || authCtx == nil {
		return nil, errortypes.NewAuthenticationError("Authentication required")
	}

	return authCtx, nil
}

//nolint:funlen // one table entry per select-option resource
func (r *Deps) SelectOptionRegistry() map[gqlmodel.SelectOptionResource]SelectOptionRegistryEntry {
	equipment := &equipmentselectoptions.Sources{
		EquipmentTypeService:         r.EquipmentTypeService,
		EquipmentManufacturerService: r.EquipmentManufacturerService,
		TrailerService:               r.TrailerService,
		TractorService:               r.TractorService,
		FleetCodeService:             r.FleetCodeService,
	}
	ediSources := &ediselectoptions.Sources{EdiService: r.EdiService}
	places := &placesselectoptions.Sources{
		UsStateService:          r.UsStateService,
		LocationService:         r.LocationService,
		RateZoneService:         r.RateZoneService,
		LocationCategoryService: r.LocationCategoryService,
		DistanceProfileService:  r.DistanceProfileService,
	}
	operations := &operationsselectoptions.Sources{
		ShipmentTypeService:         r.ShipmentTypeService,
		ServiceTypeService:          r.ServiceTypeService,
		CommodityService:            r.CommodityService,
		HazardousMaterialService:    r.HazardousMaterialService,
		OrderService:                r.OrderService,
		ShipmentService:             r.ShipmentService,
		ServiceFailureReasonCodeSvc: r.ServiceFailureReasonCodeSvc,
	}
	workforce := &workforceselectoptions.Sources{
		WorkerService:            r.WorkerService,
		SchedulingService:        r.SchedulingService,
		SelfServiceService:       r.SelfServiceService,
		OrgStructureService:      r.OrgStructureService,
		WorkerCredentialService:  r.WorkerCredentialService,
		WorkerTrainingService:    r.WorkerTrainingService,
		PerformanceReviewService: r.PerformanceReviewService,
		PtoPolicyService:         r.PtoPolicyService,
		BenefitsService:          r.BenefitsService,
	}
	identity := &identityselectoptions.Sources{
		OrganizationService: r.OrganizationService,
		UserService:         r.UserService,
		RoleService:         r.RoleService,
	}
	rating := &ratingselectoptions.Sources{
		RateMatrixService:        r.RateMatrixService,
		RateAgreementService:     r.RateAgreementService,
		AccessorialChargeService: r.AccessorialChargeService,
		DetentionPolicyService:   r.DetentionPolicyService,
		FormulaTemplateService:   r.FormulaTemplateService,
		FuelIndexRepo:            r.FuelIndexRepo,
		FuelSurchargeProgramRepo: r.FuelSurchargeProgramRepo,
	}
	accounting := &accountingselectoptions.Sources{
		FiscalYearRepo:      r.FiscalYearRepo,
		FiscalPeriodRepo:    r.FiscalPeriodRepo,
		GlAccountRepo:       r.GlAccountRepo,
		JournalEntryService: r.JournalEntryService,
		AccountTypeService:  r.AccountTypeService,
	}
	settlement := &settlementselectoptions.Sources{
		FuelPurchaseService: r.FuelPurchaseService,
		IftaService:         r.IftaService,
		DriverPayService:    r.DriverPayService,
	}
	parties := &partiesselectoptions.Sources{
		CarrierRepo:          r.CarrierRepo,
		CustomerRepo:         r.CustomerRepo,
		FactoringCompanyRepo: r.FactoringCompanyRepo,
	}
	documents := &documentsselectoptions.Sources{
		CaptureService:      r.CaptureService,
		DocumentTypeService: r.DocumentTypeService,
		EmailService:        r.EmailService,
	}
	agent := &agentselectoptions.Sources{Core: r.Core, AiProviderService: r.AiProviderService}

	return map[gqlmodel.SelectOptionResource]SelectOptionRegistryEntry{
		gqlmodel.SelectOptionResourceCarrier: {
			Resolve: parties.Carrier,
		},
		gqlmodel.SelectOptionResourceCustomer: {
			Resolve: parties.Customer,
		},
		gqlmodel.SelectOptionResourceEDIConnection: {
			Resolve: ediSources.EDIConnection,
		},
		gqlmodel.SelectOptionResourceEquipmentType: {
			Resolve: equipment.EquipmentType,
		},
		gqlmodel.SelectOptionResourceEquipmentManufacturer: {
			Resolve: equipment.EquipmentManufacturer,
		},
		gqlmodel.SelectOptionResourceTrailer: {
			Resolve: equipment.Trailer,
		},
		gqlmodel.SelectOptionResourceTractor: {
			Resolve: equipment.Tractor,
		},
		gqlmodel.SelectOptionResourceFuelCard: {
			Resolve: settlement.FuelCard,
		},
		gqlmodel.SelectOptionResourceWorker: {
			Resolve: workforce.Worker,
		},
		gqlmodel.SelectOptionResourceUsState: {
			Resolve: places.USState,
		},
		gqlmodel.SelectOptionResourceShipment: {
			Resolve: operations.Shipment,
		},
		gqlmodel.SelectOptionResourceOrder: {
			Resolve: operations.Order,
		},
		gqlmodel.SelectOptionResourceEDITransfer: {
			Resolve: ediSources.EDITransfer,
		},
		gqlmodel.SelectOptionResourceFuelIndex: {
			Resolve: rating.FuelIndex,
		},
		gqlmodel.SelectOptionResourceFuelSurchargeProgram: {
			Resolve: rating.FuelSurchargeProgram,
		},
		gqlmodel.SelectOptionResourceFactoringCompany: {
			Resolve: parties.FactoringCompany,
		},
		gqlmodel.SelectOptionResourceFiscalYear: {
			Resolve: accounting.FiscalYear,
		},
		gqlmodel.SelectOptionResourceFiscalPeriod: {
			Resolve: accounting.FiscalPeriod,
		},
		gqlmodel.SelectOptionResourceGlAccount: {
			Resolve: accounting.GLAccount,
		},
		gqlmodel.SelectOptionResourceJournalEntry: {
			Resolve: accounting.JournalEntry,
		},
		gqlmodel.SelectOptionResourceIFTAFuelType: {
			Resolve:  settlement.IFTAFuelType,
			parseIDs: gqlid.ParseCatalogIDs,
		},
		gqlmodel.SelectOptionResourceLocation: {
			Resolve: places.Location,
		},
		gqlmodel.SelectOptionResourceRateZone: {
			Resolve: places.RateZone,
		},
		gqlmodel.SelectOptionResourceFleetCode: {
			Resolve: equipment.FleetCode,
		},
		gqlmodel.SelectOptionResourceShipmentType: {
			Resolve: operations.ShipmentType,
		},
		gqlmodel.SelectOptionResourceServiceType: {
			Resolve: operations.ServiceType,
		},
		gqlmodel.SelectOptionResourceLocationCategory: {
			Resolve: places.LocationCategory,
		},
		gqlmodel.SelectOptionResourceDistanceProfile: {
			Resolve: places.DistanceProfile,
		},
		gqlmodel.SelectOptionResourceOrganization: {
			Resolve: identity.Organization,
		},
		gqlmodel.SelectOptionResourceUser: {
			Resolve: identity.User,
		},
		gqlmodel.SelectOptionResourceRole: {
			Resolve: identity.Role,
		},
		gqlmodel.SelectOptionResourceAiProvider: {
			Resolve: agent.AIProvider,
		},
		gqlmodel.SelectOptionResourceRateMatrix: {
			Resolve: rating.RateMatrix,
		},
		gqlmodel.SelectOptionResourceRateAgreement: {
			Resolve: rating.RateAgreement,
		},
		gqlmodel.SelectOptionResourceAccessorialCharge: {
			Resolve: rating.AccessorialCharge,
		},
		gqlmodel.SelectOptionResourceAccountType: {
			Resolve: accounting.AccountType,
		},
		gqlmodel.SelectOptionResourceCommodity: {
			Resolve: operations.Commodity,
		},
		gqlmodel.SelectOptionResourceDocumentType: {
			Resolve: documents.DocumentType,
		},
		gqlmodel.SelectOptionResourceCaptureDevice: {
			Resolve: documents.CaptureDevice,
		},
		gqlmodel.SelectOptionResourceCaptureProfile: {
			Resolve: documents.CaptureProfile,
		},
		gqlmodel.SelectOptionResourceDetentionPolicy: {
			Resolve: rating.DetentionPolicy,
		},
		gqlmodel.SelectOptionResourceShiftTemplate: {
			Resolve: workforce.ShiftTemplate,
		},
		gqlmodel.SelectOptionResourceWorkerPolicy: {
			Resolve: workforce.WorkerPolicy,
		},
		gqlmodel.SelectOptionResourceJobPosition: {
			Resolve: workforce.JobPosition,
		},
		gqlmodel.SelectOptionResourceFormulaTemplate: {
			Resolve: rating.FormulaTemplate,
		},
		gqlmodel.SelectOptionResourceHazardousMaterial: {
			Resolve: operations.HazardousMaterial,
		},
		gqlmodel.SelectOptionResourceServiceFailureReasonCode: {
			Resolve: operations.ServiceFailureReasonCode,
		},
		gqlmodel.SelectOptionResourceEDICommunicationProfile: {
			Resolve: ediSources.EDICommunicationProfile,
		},
		gqlmodel.SelectOptionResourceEDIDocumentType: {
			Resolve:  ediSources.EDIDocumentType,
			parseIDs: gqlid.ParseCatalogIDs,
		},
		gqlmodel.SelectOptionResourceEDIMappingProfile: {
			Resolve: ediSources.EDIMappingProfile,
		},
		gqlmodel.SelectOptionResourceEDIPartner: {
			Resolve: ediSources.EDIPartner,
		},
		gqlmodel.SelectOptionResourceEDITransactionSet: {
			Resolve:  ediSources.EDITransactionSet,
			parseIDs: gqlid.ParseCatalogIDs,
		},
		gqlmodel.SelectOptionResourceEmailProfile: {
			Resolve: documents.EmailProfile,
		},
		gqlmodel.SelectOptionResourcePayCode: {
			Resolve: settlement.PayCode,
		},
		gqlmodel.SelectOptionResourcePayProfile: {
			Resolve: settlement.PayProfile,
		},
		gqlmodel.SelectOptionResourceWorkerCredentialType: {
			Resolve: workforce.WorkerCredentialType,
		},
		gqlmodel.SelectOptionResourceTrainingCourse: {
			Resolve: workforce.TrainingCourse,
		},
		gqlmodel.SelectOptionResourcePerformanceReviewTemplate: {
			Resolve: workforce.PerformanceReviewTemplate,
		},
		gqlmodel.SelectOptionResourcePTOPolicy: {
			Resolve: workforce.PTOPolicy,
		},
		gqlmodel.SelectOptionResourceBenefitPlan: {
			Resolve: workforce.BenefitPlan,
		},
		gqlmodel.SelectOptionResourceIFTAJurisdiction: {
			Resolve: settlement.IFTAJurisdiction,
		},
	}
}
