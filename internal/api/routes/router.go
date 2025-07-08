package routes

import (
	"time"

	"github.com/emoss08/trenova/internal/api/handlers"
	"github.com/emoss08/trenova/internal/api/handlers/accessorialcharge"
	"github.com/emoss08/trenova/internal/api/handlers/ai"
	"github.com/emoss08/trenova/internal/api/handlers/analytics"
	"github.com/emoss08/trenova/internal/api/handlers/assignment"
	"github.com/emoss08/trenova/internal/api/handlers/audit"
	authHandler "github.com/emoss08/trenova/internal/api/handlers/auth"
	"github.com/emoss08/trenova/internal/api/handlers/backup"
	"github.com/emoss08/trenova/internal/api/handlers/billingcontrol"
	"github.com/emoss08/trenova/internal/api/handlers/billingqueue"
	"github.com/emoss08/trenova/internal/api/handlers/commodity"
	"github.com/emoss08/trenova/internal/api/handlers/consolidation"
	"github.com/emoss08/trenova/internal/api/handlers/consolidationsetting"
	"github.com/emoss08/trenova/internal/api/handlers/customer"
	"github.com/emoss08/trenova/internal/api/handlers/dedicatedlane"
	"github.com/emoss08/trenova/internal/api/handlers/dedicatedlanesuggestion"
	"github.com/emoss08/trenova/internal/api/handlers/document"
	"github.com/emoss08/trenova/internal/api/handlers/documentqualityconfig"
	"github.com/emoss08/trenova/internal/api/handlers/documenttype"
	"github.com/emoss08/trenova/internal/api/handlers/equipmentmanufacturer"
	"github.com/emoss08/trenova/internal/api/handlers/equipmenttype"
	"github.com/emoss08/trenova/internal/api/handlers/favorite"
	"github.com/emoss08/trenova/internal/api/handlers/fleetcode"
	"github.com/emoss08/trenova/internal/api/handlers/hazardousmaterial"
	"github.com/emoss08/trenova/internal/api/handlers/hazmatsegregationrule"
	"github.com/emoss08/trenova/internal/api/handlers/integration"
	"github.com/emoss08/trenova/internal/api/handlers/location"
	"github.com/emoss08/trenova/internal/api/handlers/locationcategory"
	"github.com/emoss08/trenova/internal/api/handlers/notification"
	"github.com/emoss08/trenova/internal/api/handlers/notificationpreference"
	organizationHandler "github.com/emoss08/trenova/internal/api/handlers/organization"
	"github.com/emoss08/trenova/internal/api/handlers/patternconfig"
	"github.com/emoss08/trenova/internal/api/handlers/permission"
	"github.com/emoss08/trenova/internal/api/handlers/reporting"
	"github.com/emoss08/trenova/internal/api/handlers/resourceeditor"
	"github.com/emoss08/trenova/internal/api/handlers/role"
	"github.com/emoss08/trenova/internal/api/handlers/routing"
	"github.com/emoss08/trenova/internal/api/handlers/servicetype"
	"github.com/emoss08/trenova/internal/api/handlers/session"
	"github.com/emoss08/trenova/internal/api/handlers/shipment"
	"github.com/emoss08/trenova/internal/api/handlers/shipmentcontrol"
	"github.com/emoss08/trenova/internal/api/handlers/shipmentmove"
	"github.com/emoss08/trenova/internal/api/handlers/shipmenttype"
	"github.com/emoss08/trenova/internal/api/handlers/stop"
	"github.com/emoss08/trenova/internal/api/handlers/tableconfiguration"
	"github.com/emoss08/trenova/internal/api/handlers/tractor"
	"github.com/emoss08/trenova/internal/api/handlers/trailer"
	"github.com/emoss08/trenova/internal/api/handlers/user"
	"github.com/emoss08/trenova/internal/api/handlers/usstate"
	"github.com/emoss08/trenova/internal/api/handlers/websocket"
	"github.com/emoss08/trenova/internal/api/handlers/worker"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/api/server"
	"github.com/emoss08/trenova/internal/core/services/auth"
	"github.com/emoss08/trenova/internal/infrastructure/cache/redis"
	"github.com/emoss08/trenova/internal/pkg/config"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/encryptcookie"
	"github.com/gofiber/fiber/v2/middleware/favicon"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/requestid"

	"go.uber.org/fx"
)

type RouterParams struct {
	fx.In

	// Config
	Config *config.Manager

	// Logger
	Logger *logger.Logger

	// Server
	Server *server.Server

	// Redis
	Redis        *redis.Client
	ScriptLoader *redis.ScriptLoader

	// Services
	AuthService *auth.Service

	// Handlers
	OrganizationHandler            *organizationHandler.Handler
	StateHandler                   *usstate.Handler
	ErrorHandler                   *validator.ErrorHandler
	AuthHandler                    *authHandler.Handler
	AIHandler                      *ai.Handler
	UserHandler                    *user.Handler
	SessionHandler                 *session.Handler
	WorkerHandler                  *worker.Handler
	TableConfigurationHandler      *tableconfiguration.Handler
	FleetCodeHandler               *fleetcode.Handler
	DocumentQualityConfigHandler   *documentqualityconfig.Handler
	EquipmentTypeHandler           *equipmenttype.Handler
	EquipmentManufacturerHandler   *equipmentmanufacturer.Handler
	ShipmentTypeHandler            *shipmenttype.Handler
	ServiceTypeHandler             *servicetype.Handler
	HazardousMaterialHandler       *hazardousmaterial.Handler
	CommodityHandler               *commodity.Handler
	LocationCategoryHandler        *locationcategory.Handler
	ReportingHandler               *reporting.Handler
	LocationHandler                *location.Handler
	TractorHandler                 *tractor.Handler
	TrailerHandler                 *trailer.Handler
	CustomerHandler                *customer.Handler
	ConsolidationHandler           *consolidation.Handler
	ShipmentHandler                *shipment.Handler
	RoutingHandler                 *routing.Handler
	AssignmentHandler              *assignment.Handler
	ShipmentMoveHandler            *shipmentmove.Handler
	StopHandler                    *stop.Handler
	ShipmentControlHandler         *shipmentcontrol.Handler
	BillingControlHandler          *billingcontrol.Handler
	BackupHandler                  *backup.Handler
	AuditHandler                   *audit.Handler
	HazmatSegregationRuleHandler   *hazmatsegregationrule.Handler
	DocumentHandler                *document.Handler
	AccessorialChargeHandler       *accessorialcharge.Handler
	DocumentTypeHandler            *documenttype.Handler
	IntegrationHandler             *integration.Handler
	AnalyticsHandler               *analytics.Handler
	BillingQueueHandler            *billingqueue.Handler
	ResourceEditorHandler          *resourceeditor.Handler
	FavoriteHandler                *favorite.Handler
	PermissionHandler              *permission.Handler
	RoleHandler                    *role.Handler
	DedicatedLaneHandler           *dedicatedlane.Handler
	DedicatedLaneSuggestionHandler *dedicatedlanesuggestion.Handler
	PatternConfigHandler           *patternconfig.Handler
	WebSocketHandler               *websocket.Handler
	NotificationPreferenceHandler  *notificationpreference.Handler
	NotificationHandler            *notification.Handler
	MetricsHandler                 *handlers.MetricsHandler
	ConsolidationSettingHandler    *consolidationsetting.Handler
}

type Router struct {
	p       RouterParams
	app     fiber.Router
	cfg     *config.Manager
	corsCfg *config.CorsConfig
}

//nolint:gocritic // The p parameter is passed using fx.In
func NewRouter(p RouterParams) *Router {
	return &Router{
		p:       p,
		app:     p.Server.Router(),
		cfg:     p.Config,
		corsCfg: p.Config.Cors(),
	}
}

func (r *Router) Setup() {
	// API Versioning
	v1 := r.app.Group("api/v1")
	// define the rate limit middleware
	rl := middleware.NewRateLimit(middleware.RateLimitParams{
		Logger:       r.p.Logger,
		Redis:        r.p.Redis,
		ScriptLoader: r.p.ScriptLoader,
	})

	// setup the global middlewares
	r.setupMiddleware()

	// Metrics endpoint (outside API versioning for Prometheus compatibility)
	r.app.Get("/metrics", r.p.MetricsHandler.GetMetrics())

	// Health check endpoint
	r.app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":    "healthy",
			"timestamp": time.Now().Unix(),
		})
	})

	r.p.AuthHandler.RegisterRoutes(v1)
	r.setupProtectedRoutes(v1, rl)
	r.p.WebSocketHandler.RegisterRoutes(v1)
}

// setupMiddleware configures the global middleware stack
func (r *Router) setupMiddleware() {
	r.app.Use(
		favicon.New(),
		compress.New(),
		helmet.New(),
		middleware.NewLogger(r.p.Logger),
		encryptcookie.New(encryptcookie.Config{
			Key: r.cfg.Server().SecretKey,
		}),
		cors.New(cors.Config{
			AllowOrigins:     r.corsCfg.AllowedOrigins,
			AllowCredentials: r.corsCfg.AllowCredentials,
			AllowHeaders:     r.corsCfg.AllowedHeaders,
			AllowMethods:     r.corsCfg.AllowedMethods,
		}),
		requestid.New(),
	)
}

// setupProtectedRoutes configures the protected routes
func (r *Router) setupProtectedRoutes(router fiber.Router, rl *middleware.RateLimiter) {
	router.Use(middleware.NewAuthMiddleware(middleware.AuthMiddlewareParams{
		Logger: r.p.Logger,
		Config: r.cfg,
		Auth:   r.p.AuthService,
	}).Authenticate())

	// WebSocket routes (must be after auth middleware)
	// router.Use("/ws", r.p.WebSocketHandler.WebSocketUpgrade)
	// router.Get("/ws/notifications", websocket.New(r.p.WebSocketHandler.HandleWebSocket))

	// Organization
	r.p.OrganizationHandler.RegisterRoutes(router, rl)

	// US States
	r.p.StateHandler.RegisterRoutes(router, rl)

	// Users
	r.p.UserHandler.RegisterRoutes(router, rl)

	// Sessions
	r.p.SessionHandler.RegisterRoutes(router)

	// Workers
	r.p.WorkerHandler.RegisterRoutes(router, rl)

	// Table Configurations
	r.p.TableConfigurationHandler.RegisterRoutes(router, rl)

	// Fleet Codes
	r.p.FleetCodeHandler.RegisterRoutes(router, rl)

	// Document Quality Configs
	r.p.DocumentQualityConfigHandler.RegisterRoutes(router, rl)

	// Equipment Types
	r.p.EquipmentTypeHandler.RegisterRoutes(router, rl)

	// Equipment Manufacturers
	r.p.EquipmentManufacturerHandler.RegisterRoutes(router, rl)

	// Shipment Types
	r.p.ShipmentTypeHandler.RegisterRoutes(router, rl)

	// Service Types
	r.p.ServiceTypeHandler.RegisterRoutes(router, rl)

	// Hazardous Materials
	r.p.HazardousMaterialHandler.RegisterRoutes(router, rl)

	// Commodities
	r.p.CommodityHandler.RegisterRoutes(router, rl)

	// Location Categories
	r.p.LocationCategoryHandler.RegisterRoutes(router, rl)

	// Reporting
	r.p.ReportingHandler.RegisterRoutes(router, rl)

	// Locations
	r.p.LocationHandler.RegisterRoutes(router, rl)

	// Tractors
	r.p.TractorHandler.RegisterRoutes(router, rl)

	// Trailers
	r.p.TrailerHandler.RegisterRoutes(router, rl)

	// Customers
	r.p.CustomerHandler.RegisterRoutes(router, rl)

	// Consolidations
	r.p.ConsolidationHandler.RegisterRoutes(router, rl)

	// Shipments
	r.p.ShipmentHandler.RegisterRoutes(router, rl)

	// Routing
	r.p.RoutingHandler.RegisterRoutes(router, rl)

	// Assignments
	r.p.AssignmentHandler.RegisterRoutes(router, rl)

	// Shipment Moves
	r.p.ShipmentMoveHandler.RegisterRoutes(router, rl)

	// Stops
	r.p.StopHandler.RegisterRoutes(router, rl)

	// Shipment Control
	r.p.ShipmentControlHandler.RegisterRoutes(router, rl)

	// Backup
	r.p.BackupHandler.RegisterRoutes(router, rl)

	// Audit Logs
	r.p.AuditHandler.RegisterRoutes(router, rl)

	// Hazmat Segregation Rules
	r.p.HazmatSegregationRuleHandler.RegisterRoutes(router, rl)

	// Documents
	r.p.DocumentHandler.RegisterRoutes(router, rl)

	// Billing Control
	r.p.BillingControlHandler.RegisterRoutes(router, rl)

	// Accessorial Charges
	r.p.AccessorialChargeHandler.RegisterRoutes(router, rl)

	// Document Types
	r.p.DocumentTypeHandler.RegisterRoutes(router, rl)

	// Integrations
	r.p.IntegrationHandler.RegisterRoutes(router, rl)

	// Analytics
	r.p.AnalyticsHandler.RegisterRoutes(router, rl)

	// Billing Queue
	r.p.BillingQueueHandler.RegisterRoutes(router, rl)

	// Resource Editor
	r.p.ResourceEditorHandler.RegisterRoutes(router, rl)

	// Favorites
	r.p.FavoriteHandler.RegisterRoutes(router, rl)

	// Permissions
	r.p.PermissionHandler.RegisterRoutes(router, rl)

	// Roles
	r.p.RoleHandler.RegisterRoutes(router, rl)

	// Dedicated Lanes
	r.p.DedicatedLaneHandler.RegisterRoutes(router, rl)

	// Dedicated Lane Suggestions
	r.p.DedicatedLaneSuggestionHandler.RegisterRoutes(router, rl)

	// Pattern Config
	r.p.PatternConfigHandler.RegisterRoutes(router, rl)

	// Notification Preferences
	r.p.NotificationPreferenceHandler.RegisterRoutes(router, rl)

	// Notifications
	r.p.NotificationHandler.RegisterRoutes(router, rl)

	// Consolidation Settings
	r.p.ConsolidationSettingHandler.RegisterRoutes(router, rl)

	// AI Classification
	r.p.AIHandler.RegisterRoutes(router, rl)
}
