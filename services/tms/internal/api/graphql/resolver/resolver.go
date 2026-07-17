package resolver

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/accessorialchargeservice"
	"github.com/emoss08/trenova/internal/core/services/accounttypeservice"
	"github.com/emoss08/trenova/internal/core/services/apikeyservice"
	"github.com/emoss08/trenova/internal/core/services/commodityservice"
	"github.com/emoss08/trenova/internal/core/services/customerservice"
	"github.com/emoss08/trenova/internal/core/services/customfieldservice"
	"github.com/emoss08/trenova/internal/core/services/distanceoverrideservice"
	"github.com/emoss08/trenova/internal/core/services/distanceprofileservice"
	"github.com/emoss08/trenova/internal/core/services/documentpacketruleservice"
	"github.com/emoss08/trenova/internal/core/services/documenttypeservice"
	"github.com/emoss08/trenova/internal/core/services/ediinboundservice"
	"github.com/emoss08/trenova/internal/core/services/ediservice"
	"github.com/emoss08/trenova/internal/core/services/emailservice"
	"github.com/emoss08/trenova/internal/core/services/equipmentmanufacturerservice"
	"github.com/emoss08/trenova/internal/core/services/equipmenttypeservice"
	"github.com/emoss08/trenova/internal/core/services/fiscalyearservice"
	"github.com/emoss08/trenova/internal/core/services/fleetcodeservice"
	"github.com/emoss08/trenova/internal/core/services/formulatemplateservice"
	"github.com/emoss08/trenova/internal/core/services/hazardousmaterialservice"
	"github.com/emoss08/trenova/internal/core/services/hazmatsegregationruleservice"
	"github.com/emoss08/trenova/internal/core/services/holdreasonservice"
	"github.com/emoss08/trenova/internal/core/services/journalreversalservice"
	"github.com/emoss08/trenova/internal/core/services/locationcategoryservice"
	"github.com/emoss08/trenova/internal/core/services/locationservice"
	"github.com/emoss08/trenova/internal/core/services/manualjournalservice"
	"github.com/emoss08/trenova/internal/core/services/notificationservice"
	"github.com/emoss08/trenova/internal/core/services/orderservice"
	"github.com/emoss08/trenova/internal/core/services/ratetableservice"
	reportingservice "github.com/emoss08/trenova/internal/core/services/reporting"
	"github.com/emoss08/trenova/internal/core/services/roleservice"
	"github.com/emoss08/trenova/internal/core/services/servicetypeservice"
	"github.com/emoss08/trenova/internal/core/services/shipmenttypeservice"
	"github.com/emoss08/trenova/internal/core/services/sidebarpreferenceservice"
	"github.com/emoss08/trenova/internal/core/services/storedmileageservice"
	"github.com/emoss08/trenova/internal/core/services/tablechangealertservice"
	"github.com/emoss08/trenova/internal/core/services/tableconfigurationservice"
	"github.com/emoss08/trenova/internal/core/services/tractorservice"
	"github.com/emoss08/trenova/internal/core/services/trailerservice"
	"github.com/emoss08/trenova/internal/core/services/userservice"
	"github.com/emoss08/trenova/internal/core/services/usstateservice"
	"github.com/emoss08/trenova/internal/core/services/workerptoservice"
	"github.com/emoss08/trenova/internal/core/services/workerservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger                       *zap.Logger
	AnalyticsService             services.AnalyticsService
	OrganizationService          services.OrganizationService
	ShipmentService              services.ShipmentService
	ShipmentCommentService       services.ShipmentCommentService
	ShipmentEventService         services.ShipmentEventService
	ShipmentImportAssistant      services.ShipmentImportAssistantService `optional:"true"`
	EquipmentManufacturerService *equipmentmanufacturerservice.Service
	EDIService                   *ediservice.Service
	EDIInboundService            *ediinboundservice.Service
	EquipmentTypeService         *equipmenttypeservice.Service
	AccessorialChargeService     *accessorialchargeservice.Service
	AccountTypeService           *accounttypeservice.Service
	CommodityService             *commodityservice.Service
	CustomerService              *customerservice.Service
	CustomFieldService           *customfieldservice.Service
	FleetCodeService             *fleetcodeservice.Service
	HazardousMaterialService     *hazardousmaterialservice.Service
	HazmatSegregationRuleService *hazmatsegregationruleservice.Service
	HoldReasonService            *holdreasonservice.Service
	LocationService              *locationservice.Service
	LocationCategoryService      *locationcategoryservice.Service
	DocumentTypeService          *documenttypeservice.Service
	ServiceTypeService           *servicetypeservice.Service
	OrderService                 *orderservice.Service
	ShipmentTypeService          *shipmenttypeservice.Service
	TractorService               *tractorservice.Service
	TrailerService               *trailerservice.Service
	USStateService               *usstateservice.Service
	WorkerService                *workerservice.Service
	WorkerPTOService             *workerptoservice.Service
	FiscalYearService            *fiscalyearservice.Service
	FormulaTemplateService       *formulatemplateservice.Service
	RateTableService             *ratetableservice.Service
	EmailService                 *emailservice.Service
	DocumentPacketRuleService    *documentpacketruleservice.Service
	DistanceOverrideService      *distanceoverrideservice.Service
	DistanceProfileService       *distanceprofileservice.Service
	StoredMileageService         *storedmileageservice.Service
	ManualJournalService         *manualjournalservice.Service
	JournalReversalService       *journalreversalservice.Service
	AuditService                 services.AuditService
	ServiceFailureReasonCodeSvc  services.ServiceFailureReasonCodeService
	ServiceFailureSvc            services.ServiceFailureService
	BillingQueueService          services.BillingQueueService
	InvoiceService               services.InvoiceService
	InvoiceAdjustmentService     services.InvoiceAdjustmentService
	IAMService                   services.IAMService
	RoleService                  *roleservice.Service
	UserService                  *userservice.Service
	APIKeyService                *apikeyservice.Service
	TableChangeAlertService      *tablechangealertservice.Service
	TableConfigurationService    *tableconfigurationservice.Service
	SidebarPreferenceService     *sidebarpreferenceservice.Service
	ReportingService             *reportingservice.Service
	NotificationService          *notificationservice.Service
	PermissionEngine             services.PermissionEngine
}

type Resolver struct {
	l                            *zap.Logger
	analyticsService             services.AnalyticsService
	organizationService          services.OrganizationService
	shipmentService              services.ShipmentService
	shipmentCommentService       services.ShipmentCommentService
	shipmentEventService         services.ShipmentEventService
	shipmentImportAssistant      services.ShipmentImportAssistantService
	ediService                   *ediservice.Service
	ediInboundService            *ediinboundservice.Service
	equipmentTypeService         *equipmenttypeservice.Service
	accessorialChargeService     *accessorialchargeservice.Service
	accountTypeService           *accounttypeservice.Service
	commodityService             *commodityservice.Service
	customerService              *customerservice.Service
	customFieldService           *customfieldservice.Service
	fleetCodeService             *fleetcodeservice.Service
	hazardousMaterialService     *hazardousmaterialservice.Service
	hazmatSegregationRuleService *hazmatsegregationruleservice.Service
	holdReasonService            *holdreasonservice.Service
	locationService              *locationservice.Service
	locationCategoryService      *locationcategoryservice.Service
	documentTypeService          *documenttypeservice.Service
	serviceTypeService           *servicetypeservice.Service
	orderService                 *orderservice.Service
	shipmentTypeService          *shipmenttypeservice.Service
	equipmentManufacturerService *equipmentmanufacturerservice.Service
	tractorService               *tractorservice.Service
	trailerService               *trailerservice.Service
	usStateService               *usstateservice.Service
	workerService                *workerservice.Service
	workerPTOService             *workerptoservice.Service
	fiscalYearService            *fiscalyearservice.Service
	formulaTemplateService       *formulatemplateservice.Service
	rateTableService             *ratetableservice.Service
	emailService                 *emailservice.Service
	documentPacketRuleService    *documentpacketruleservice.Service
	distanceOverrideService      *distanceoverrideservice.Service
	distanceProfileService       *distanceprofileservice.Service
	storedMileageService         *storedmileageservice.Service
	manualJournalService         *manualjournalservice.Service
	journalReversalService       *journalreversalservice.Service
	auditService                 services.AuditService
	serviceFailureReasonCodeSvc  services.ServiceFailureReasonCodeService
	serviceFailureSvc            services.ServiceFailureService
	billingQueueService          services.BillingQueueService
	invoiceService               services.InvoiceService
	invoiceAdjustmentService     services.InvoiceAdjustmentService
	iamService                   services.IAMService
	roleService                  *roleservice.Service
	userService                  *userservice.Service
	apiKeyService                *apikeyservice.Service
	tableChangeAlertService      *tablechangealertservice.Service
	tableConfigurationService    *tableconfigurationservice.Service
	sidebarPreferenceService     *sidebarpreferenceservice.Service
	notificationService          *notificationservice.Service
	reportingService             *reportingservice.Service
	permissionEngine             services.PermissionEngine
}

func New(p Params) *Resolver {
	return &Resolver{
		l:                            p.Logger.Named("api.graphql.resolver"),
		analyticsService:             p.AnalyticsService,
		organizationService:          p.OrganizationService,
		shipmentService:              p.ShipmentService,
		shipmentCommentService:       p.ShipmentCommentService,
		shipmentEventService:         p.ShipmentEventService,
		shipmentImportAssistant:      p.ShipmentImportAssistant,
		ediService:                   p.EDIService,
		ediInboundService:            p.EDIInboundService,
		equipmentTypeService:         p.EquipmentTypeService,
		accessorialChargeService:     p.AccessorialChargeService,
		accountTypeService:           p.AccountTypeService,
		commodityService:             p.CommodityService,
		customerService:              p.CustomerService,
		customFieldService:           p.CustomFieldService,
		fleetCodeService:             p.FleetCodeService,
		hazardousMaterialService:     p.HazardousMaterialService,
		hazmatSegregationRuleService: p.HazmatSegregationRuleService,
		holdReasonService:            p.HoldReasonService,
		locationService:              p.LocationService,
		locationCategoryService:      p.LocationCategoryService,
		documentTypeService:          p.DocumentTypeService,
		serviceTypeService:           p.ServiceTypeService,
		orderService:                 p.OrderService,
		shipmentTypeService:          p.ShipmentTypeService,
		equipmentManufacturerService: p.EquipmentManufacturerService,
		tractorService:               p.TractorService,
		trailerService:               p.TrailerService,
		usStateService:               p.USStateService,
		workerService:                p.WorkerService,
		workerPTOService:             p.WorkerPTOService,
		fiscalYearService:            p.FiscalYearService,
		formulaTemplateService:       p.FormulaTemplateService,
		rateTableService:             p.RateTableService,
		emailService:                 p.EmailService,
		documentPacketRuleService:    p.DocumentPacketRuleService,
		distanceOverrideService:      p.DistanceOverrideService,
		distanceProfileService:       p.DistanceProfileService,
		storedMileageService:         p.StoredMileageService,
		manualJournalService:         p.ManualJournalService,
		journalReversalService:       p.JournalReversalService,
		auditService:                 p.AuditService,
		serviceFailureReasonCodeSvc:  p.ServiceFailureReasonCodeSvc,
		serviceFailureSvc:            p.ServiceFailureSvc,
		billingQueueService:          p.BillingQueueService,
		invoiceService:               p.InvoiceService,
		invoiceAdjustmentService:     p.InvoiceAdjustmentService,
		iamService:                   p.IAMService,
		roleService:                  p.RoleService,
		userService:                  p.UserService,
		apiKeyService:                p.APIKeyService,
		tableChangeAlertService:      p.TableChangeAlertService,
		tableConfigurationService:    p.TableConfigurationService,
		sidebarPreferenceService:     p.SidebarPreferenceService,
		notificationService:          p.NotificationService,
		reportingService:             p.ReportingService,
		permissionEngine:             p.PermissionEngine,
	}
}

func (r *Resolver) requirePermission(
	ctx context.Context,
	resource permission.Resource,
	operation permission.Operation,
) (*authctx.AuthContext, error) {
	authCtx, ok := gqlctx.AuthContext(ctx)
	if !ok || authCtx == nil {
		return nil, errortypes.NewAuthenticationError("Authentication required")
	}

	result, err := r.permissionEngine.Check(
		ctx,
		middleware.BuildPermissionCheckRequest(authCtx, resource.String(), operation),
	)
	if err != nil {
		return nil, err
	}
	if !result.Allowed {
		return nil, errortypes.NewAuthorizationError(
			fmt.Sprintf(
				"You don't have permission to perform this action: %s %s",
				resource,
				operation,
			),
		)
	}

	return authCtx, nil
}

func (r *Resolver) hasPermission(
	ctx context.Context,
	authCtx *authctx.AuthContext,
	resource permission.Resource,
	operation permission.Operation,
) bool {
	result, err := r.permissionEngine.Check(
		ctx,
		middleware.BuildPermissionCheckRequest(authCtx, resource.String(), operation),
	)
	if err != nil {
		r.l.Warn("permission check failed",
			zap.String("resource", resource.String()),
			zap.Error(err))
		return false
	}

	return result.Allowed
}

func (r *Resolver) requireAuth(ctx context.Context) (*authctx.AuthContext, error) {
	authCtx, ok := gqlctx.AuthContext(ctx)
	if !ok || authCtx == nil {
		return nil, errortypes.NewAuthenticationError("Authentication required")
	}

	return authCtx, nil
}

func tenantInfo(authCtx *authctx.AuthContext) pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}
}
