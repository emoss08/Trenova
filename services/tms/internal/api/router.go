package api

import (
	"net/http"
	"time"

	"github.com/emoss08/trenova/internal/api/handlers"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/observability"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/fx"
)

type RouterParams struct {
	fx.In

	Server                         *Server
	Config                         *config.Config
	Middleware                     *observability.Middleware
	Metrics                        *observability.MetricsRegistry
	AuthMiddleware                 *middleware.AuthMiddleware
	AuthHandler                    *handlers.AuthHandler
	APITokenHandler                *handlers.APITokenHandler
	BillingControlHandler          *handlers.BillingControlHandler
	DedicatedLaneSuggestionHandler *handlers.DedicatedLaneSuggestionHandler
	VariableHandler                *handlers.VariableHandler
	DocumentTypeHandler            *handlers.DocumentTypeHandler
	EmailProfileHandler            *handlers.EmailProfileHandler
	AccessorialChargeHandler       *handlers.AccessorialChargeHandler
	PermissionHandler              *handlers.PermissionHandler
	CustomerHandler                *handlers.CustomerHandler
	EquipmentTypeHandler           *handlers.EquipmentTypeHandler
	EquipmentManufacturerHandler   *handlers.EquipmentManufacturerHandler
	HazmatSegregationRuleHandler   *handlers.HazmatSegregationRuleHandler
	AILogHandler                   *handlers.AILogHandler
	GoogleMapsHandler              *handlers.GoogleMapsHandler
	OrganizationHandler            *handlers.OrganizationHandler
	FiscalYearHandler              *handlers.FiscalYearHandler
	FiscalPeriodHandler            *handlers.FiscalPeriodHandler
	GLAccountHandler               *handlers.GLAccountHandler
	JournalEntryHandler            *handlers.JournalEntryHandler
	TrailerHandler                 *handlers.TrailerHandler
	WorkerHandler                  *handlers.WorkerHandler
	ShipmentControlHandler         *handlers.ShipmentControlHandler
	ShipmentTypeHandler            *handlers.ShipmentTypeHandler
	TractorHandler                 *handlers.TractorHandler
	ServiceTypeHandler             *handlers.ServiceTypeHandler
	AuditHandler                   *handlers.AuditHandler
	CommodityHandler               *handlers.CommodityHandler
	TableConfigurationHandler      *handlers.TableConfigurationHandler
	HoldReasonHandler              *handlers.HoldReasonHandler
	HazardousMaterialHandler       *handlers.HazardousMaterialHandler
	LocationCategoryHandler        *handlers.LocationCategoryHandler
	FleetCodeHandler               *handlers.FleetCodeHandler
	LocationHandler                *handlers.LocationHandler
	ShipmentHandler                *handlers.ShipmentHandler
	DedicatedLaneHandler           *handlers.DedicatedLaneHandler
	DistanceOverrideHandler        *handlers.DistanceOverrideHandler
	UserHandler                    *handlers.UserHandler
	UsStateHandler                 *handlers.UsStateHandler
	WebSocketHandler               *handlers.WebSocketHandler
	NotificationHandler            *handlers.NotificationHandler
	SearchHandler                  *handlers.SearchHandler
	PageFavoriteHandler            *handlers.PageFavoriteHandler
	DispatchControlHandler         *handlers.DispatchControlHandler
	AccountTypeHandler             *handlers.AccountTypeHandler
	DataRetentionHandler           *handlers.DataRetentionHandler
	ClassificationHandler          *handlers.ClassificationHandler
	PatternConfigHandler           *handlers.PatternConfigHandler
	AccountingControlHandler       *handlers.AccountingControlHandler
	UserPreferenceHandler          *handlers.UserPreferenceHandler
	ReportHandler                  *handlers.ReportHandler
	WorkflowHandler                *handlers.WorkflowHandler
	WorkflowExecutionHandler       *handlers.WorkflowExecutionHandler
	WorkflowTemplateHandler        *handlers.WorkflowTemplateHandler
	ErrorHandler                   *helpers.ErrorHandler
}

type Router struct {
	s                              *Server
	cfg                            *config.Config
	mw                             *observability.Middleware
	authMw                         *middleware.AuthMiddleware
	authHandler                    *handlers.AuthHandler
	apiTokenHandler                *handlers.APITokenHandler
	ailogHandler                   *handlers.AILogHandler
	variableHandler                *handlers.VariableHandler
	documentTypeHandler            *handlers.DocumentTypeHandler
	accessorialChargeHandler       *handlers.AccessorialChargeHandler
	equipmentTypeHandler           *handlers.EquipmentTypeHandler
	workerHandler                  *handlers.WorkerHandler
	dedicatedLaneSuggestionHandler *handlers.DedicatedLaneSuggestionHandler
	trailerHandler                 *handlers.TrailerHandler
	distanceOverrideHandler        *handlers.DistanceOverrideHandler
	equipmentManufacturerHandler   *handlers.EquipmentManufacturerHandler
	accountTypeHandler             *handlers.AccountTypeHandler
	fleetCodeHandler               *handlers.FleetCodeHandler
	tractorHandler                 *handlers.TractorHandler
	customerHandler                *handlers.CustomerHandler
	hazmatSegregationRuleHandler   *handlers.HazmatSegregationRuleHandler
	permissionHandler              *handlers.PermissionHandler
	organizationHandler            *handlers.OrganizationHandler
	fiscalYearHandler              *handlers.FiscalYearHandler
	fiscalPeriodHandler            *handlers.FiscalPeriodHandler
	glAccountHandler               *handlers.GLAccountHandler
	journalEntryHandler            *handlers.JournalEntryHandler
	serviceTypeHandler             *handlers.ServiceTypeHandler
	commodityHandler               *handlers.CommodityHandler
	googleMapsHandler              *handlers.GoogleMapsHandler
	shipmentControlHandler         *handlers.ShipmentControlHandler
	dispatchControlHandler         *handlers.DispatchControlHandler
	shipmentTypeHandler            *handlers.ShipmentTypeHandler
	emailProfileHandler            *handlers.EmailProfileHandler
	billingControlHandler          *handlers.BillingControlHandler
	holdReasonHandler              *handlers.HoldReasonHandler
	hazardousMaterialHandler       *handlers.HazardousMaterialHandler
	locationCategoryHandler        *handlers.LocationCategoryHandler
	locationHandler                *handlers.LocationHandler
	shipmentHandler                *handlers.ShipmentHandler
	dedicatedLaneHandler           *handlers.DedicatedLaneHandler
	tableConfigurationHandler      *handlers.TableConfigurationHandler
	auditHandler                   *handlers.AuditHandler
	usStateHandler                 *handlers.UsStateHandler
	websocketHandler               *handlers.WebSocketHandler
	notificationHandler            *handlers.NotificationHandler
	searchHandler                  *handlers.SearchHandler
	pageFavoriteHandler            *handlers.PageFavoriteHandler
	dataRetentionHandler           *handlers.DataRetentionHandler
	classificationHandler          *handlers.ClassificationHandler
	patternConfigHandler           *handlers.PatternConfigHandler
	accountingControlHandler       *handlers.AccountingControlHandler
	userPreferenceHandler          *handlers.UserPreferenceHandler
	reportHandler                  *handlers.ReportHandler
	workflowHandler                *handlers.WorkflowHandler
	workflowExecutionHandler       *handlers.WorkflowExecutionHandler
	workflowTemplateHandler        *handlers.WorkflowTemplateHandler
	errorHandler                   *helpers.ErrorHandler
	userHandler                    *handlers.UserHandler
	metrics                        *observability.MetricsRegistry
}

//nolint:gocritic // Ignore the large number of parameters for the router
func NewRouter(p RouterParams) *Router {
	return &Router{
		s:                              p.Server,
		cfg:                            p.Config,
		mw:                             p.Middleware,
		authMw:                         p.AuthMiddleware,
		authHandler:                    p.AuthHandler,
		apiTokenHandler:                p.APITokenHandler,
		ailogHandler:                   p.AILogHandler,
		variableHandler:                p.VariableHandler,
		emailProfileHandler:            p.EmailProfileHandler,
		organizationHandler:            p.OrganizationHandler,
		permissionHandler:              p.PermissionHandler,
		accessorialChargeHandler:       p.AccessorialChargeHandler,
		workerHandler:                  p.WorkerHandler,
		commodityHandler:               p.CommodityHandler,
		fiscalYearHandler:              p.FiscalYearHandler,
		fiscalPeriodHandler:            p.FiscalPeriodHandler,
		glAccountHandler:               p.GLAccountHandler,
		journalEntryHandler:            p.JournalEntryHandler,
		googleMapsHandler:              p.GoogleMapsHandler,
		hazmatSegregationRuleHandler:   p.HazmatSegregationRuleHandler,
		customerHandler:                p.CustomerHandler,
		accountTypeHandler:             p.AccountTypeHandler,
		shipmentControlHandler:         p.ShipmentControlHandler,
		dispatchControlHandler:         p.DispatchControlHandler,
		shipmentTypeHandler:            p.ShipmentTypeHandler,
		shipmentHandler:                p.ShipmentHandler,
		dedicatedLaneHandler:           p.DedicatedLaneHandler,
		dedicatedLaneSuggestionHandler: p.DedicatedLaneSuggestionHandler,
		equipmentTypeHandler:           p.EquipmentTypeHandler,
		equipmentManufacturerHandler:   p.EquipmentManufacturerHandler,
		fleetCodeHandler:               p.FleetCodeHandler,
		serviceTypeHandler:             p.ServiceTypeHandler,
		searchHandler:                  p.SearchHandler,
		tractorHandler:                 p.TractorHandler,
		tableConfigurationHandler:      p.TableConfigurationHandler,
		hazardousMaterialHandler:       p.HazardousMaterialHandler,
		distanceOverrideHandler:        p.DistanceOverrideHandler,
		documentTypeHandler:            p.DocumentTypeHandler,
		trailerHandler:                 p.TrailerHandler,
		holdReasonHandler:              p.HoldReasonHandler,
		dataRetentionHandler:           p.DataRetentionHandler,
		locationHandler:                p.LocationHandler,
		billingControlHandler:          p.BillingControlHandler,
		auditHandler:                   p.AuditHandler,
		userHandler:                    p.UserHandler,
		usStateHandler:                 p.UsStateHandler,
		websocketHandler:               p.WebSocketHandler,
		notificationHandler:            p.NotificationHandler,
		pageFavoriteHandler:            p.PageFavoriteHandler,
		locationCategoryHandler:        p.LocationCategoryHandler,
		classificationHandler:          p.ClassificationHandler,
		patternConfigHandler:           p.PatternConfigHandler,
		accountingControlHandler:       p.AccountingControlHandler,
		userPreferenceHandler:          p.UserPreferenceHandler,
		reportHandler:                  p.ReportHandler,
		workflowHandler:                p.WorkflowHandler,
		workflowExecutionHandler:       p.WorkflowExecutionHandler,
		workflowTemplateHandler:        p.WorkflowTemplateHandler,
		errorHandler:                   p.ErrorHandler,
		metrics:                        p.Metrics,
	}
}

func (r *Router) Setup() {
	r.setupMiddleware()
	r.setupRoutes()
}

func (r *Router) setupMiddleware() {
	r.setupCORS()

	r.s.router.Use(r.errorHandler.Middleware())
	r.s.router.Use(r.mw.TracingMiddleware())
	r.s.router.Use(gin.Logger())
	r.s.router.Use(gin.Recovery())
	r.s.router.Use(requestid.New())
	r.s.router.Use(
		gzip.Gzip(gzip.DefaultCompression, gzip.WithExcludedPaths([]string{"/metrics", "/health"})),
	)
}

func (r *Router) setupCORS() {
	if !r.cfg.Server.CORS.Enabled {
		return
	}

	corsConfig := cors.Config{
		AllowOrigins:     r.cfg.Server.CORS.AllowedOrigins,
		AllowMethods:     r.cfg.Server.CORS.AllowedMethods,
		AllowHeaders:     r.cfg.Server.CORS.AllowedHeaders,
		ExposeHeaders:    r.cfg.Server.CORS.ExposeHeaders,
		AllowCredentials: r.cfg.Server.CORS.Credentials,
		MaxAge:           time.Duration(r.cfg.Server.CORS.MaxAge) * time.Second,
	}

	// If AllowOrigins is empty or contains "*", allow all origins with credentials disabled
	if len(corsConfig.AllowOrigins) == 0 ||
		(len(corsConfig.AllowOrigins) == 1 && corsConfig.AllowOrigins[0] == "*") {
		corsConfig.AllowAllOrigins = true
		corsConfig.AllowOrigins = nil
		corsConfig.AllowCredentials = false
	}

	r.s.router.Use(cors.New(corsConfig))
}

func (r *Router) setupRoutes() {
	// Health and metrics endpoints (no auth required)
	r.s.router.GET("/health", r.healthCheck)

	// Metrics endpoint for Prometheus scraping
	if r.metrics.IsEnabled() {
		r.s.router.GET("/metrics", r.metricsHandler)
	}

	// API v1 routes
	v1 := r.s.router.Group("/api/v1")
	// Public routes (no auth required)
	r.setupPublicRoutes(v1)

	// Protected routes (auth required)
	r.setupProtectedRoutes(v1)
}

func (r *Router) setupPublicRoutes(rg *gin.RouterGroup) {
	rg.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong"})
	})

	r.authHandler.RegisterPublicRoutes(rg)
}

func (r *Router) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "trenova-api",
		"version": r.cfg.App.Version,
	})
}

func (r *Router) metricsHandler(c *gin.Context) {
	handler := promhttp.HandlerFor(
		r.metrics.Registry(),
		promhttp.HandlerOpts{
			EnableOpenMetrics: true,
		},
	)
	handler.ServeHTTP(c.Writer, c.Request)
}

func (r *Router) setupProtectedRoutes(rg *gin.RouterGroup) {
	protected := rg.Group("")
	protected.Use(r.authMw.RequireAuth())

	r.apiTokenHandler.RegisterRoutes(protected)
	r.organizationHandler.RegisterRoutes(protected)
	r.auditHandler.RegisterRoutes(protected)
	r.userHandler.RegisterRoutes(protected)
	r.usStateHandler.RegisterRoutes(protected)
	r.websocketHandler.RegisterRoutes(protected)
	r.notificationHandler.RegisterRoutes(protected)
	r.pageFavoriteHandler.RegisterRoutes(protected)
	r.tableConfigurationHandler.RegisterRoutes(protected)
	r.holdReasonHandler.RegisterRoutes(protected)
	r.hazardousMaterialHandler.RegisterRoutes(protected)
	r.hazmatSegregationRuleHandler.RegisterRoutes(protected)
	r.billingControlHandler.RegisterRoutes(protected)
	r.shipmentControlHandler.RegisterRoutes(protected)
	r.dataRetentionHandler.RegisterRoutes(protected)
	r.locationCategoryHandler.RegisterRoutes(protected)
	r.locationHandler.RegisterRoutes(protected)
	r.fleetCodeHandler.RegisterRoutes(protected)
	r.emailProfileHandler.RegisterRoutes(protected)
	r.documentTypeHandler.RegisterRoutes(protected)
	r.equipmentTypeHandler.RegisterRoutes(protected)
	r.equipmentManufacturerHandler.RegisterRoutes(protected)
	r.workerHandler.RegisterRoutes(protected)
	r.shipmentTypeHandler.RegisterRoutes(protected)
	r.serviceTypeHandler.RegisterRoutes(protected)
	r.tractorHandler.RegisterRoutes(protected)
	r.trailerHandler.RegisterRoutes(protected)
	r.permissionHandler.RegisterRoutes(protected)
	r.commodityHandler.RegisterRoutes(protected)
	r.classificationHandler.RegisterRoutes(protected)
	r.ailogHandler.RegisterRoutes(protected)
	r.distanceOverrideHandler.RegisterRoutes(protected)
	r.googleMapsHandler.RegisterRoutes(protected)
	r.customerHandler.RegisterRoutes(protected)
	r.variableHandler.RegisterRoutes(protected)
	r.accessorialChargeHandler.RegisterRoutes(protected)
	r.shipmentHandler.RegisterRoutes(protected)
	r.dedicatedLaneHandler.RegisterRoutes(protected)
	r.dedicatedLaneSuggestionHandler.RegisterRoutes(protected)
	r.patternConfigHandler.RegisterRoutes(protected)
	r.dispatchControlHandler.RegisterRoutes(protected)
	r.searchHandler.RegisterRoutes(protected)
	r.accountTypeHandler.RegisterRoutes(protected)
	r.fiscalYearHandler.RegisterRoutes(protected)
	r.fiscalPeriodHandler.RegisterRoutes(protected)
	r.glAccountHandler.RegisterRoutes(protected)
	r.journalEntryHandler.RegisterRoutes(protected)
	r.accountingControlHandler.RegisterRoutes(protected)
	r.userPreferenceHandler.RegisterRoutes(protected)
	r.reportHandler.RegisterRoutes(protected)
	r.workflowHandler.RegisterRoutes(protected)
	r.workflowExecutionHandler.RegisterRoutes(protected)
	r.workflowTemplateHandler.RegisterRoutes(protected)
}
