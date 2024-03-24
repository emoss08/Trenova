package routes

import (
	"github.com/emoss08/trenova/middleware"
	"github.com/go-chi/chi/v5"
	chi_middleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func InitializeRouter() *chi.Mux {
	r := chi.NewRouter()

	// logger := &httpretty.Logger{
	// 	Time:           true,
	// 	TLS:            true,
	// 	RequestHeader:  true,
	// 	RequestBody:    true,
	// 	ResponseHeader: true,
	// 	ResponseBody:   true,
	// 	Colors:         true,
	// }

	r.Use(chi_middleware.RequestID)
	r.Use(chi_middleware.RealIP)
	r.Use(chi_middleware.Logger)
	r.Use(chi_middleware.Compress(5))
	r.Use(chi_middleware.Recoverer)
	r.Use(chi_middleware.StripSlashes)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://localhost:5173", "http://localhost:5173"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-Idempotency-Key"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Route("/api", func(r chi.Router) {
		// Register Auth Routes
		registerAuthRoutes(r)

		// Protected Routes
		r.Group(func(r chi.Router) {
			r.Use(middleware.SessionMiddleware)
			r.Use(middleware.IdempotencyMiddleware)

			// Register Organization Routes
			registerOrganizationRoutes(r)

			// Register Billing Control Routes
			registerBillingControlRouter(r)

			// Register Accounting Control Routes
			registerAccountingControlRouter(r)

			// Register User Routes
			registerUserRoutes(r)

			// Register Invoice Control Routes
			registerInvoiceControlRouter(r)

			// Reigster Shipment Control Routes
			registerShipmentControlRouter(r)

			// Register Dispatch Control Routes
			registerDispatchControlRouter(r)

			// Register Feasibility Tool Control Routes
			registerFeasibilityControlRouter(r)

			// Register Route Control Routes
			registerRouteControlRouter(r)

			// Register Email control Routes
			registerEmailControlRoutes(r)

			// Register Email Profile Routes
			registerEmailProfileRouter(r)

			// Register User Favorites Routes
			registerUserFavoritesRoutes(r)

			// Register Revenue Code Routes
			registerRevenueCodeRouter(r)

			// Register General Ledger Account Routes
			registerGLAccountRoutes(r)

			// Register Commodity Routes
			registerCommodityRoutes(r)

			// Register Hazardous Material Routes
			registerHazardousMaterialRouter(r)

			// Rgister Charge Type Routes
			registerChargeTypeRouter(r)

			// Register Division Code Routes
			registerDivisionCodeRouter(r)

			// Register Accessorial Charge Routes
			registerAccessorialChargeRouter(r)

			// Register Customer Routes
			registerCustomerRouter(r)

			// Register US State Routes
			registerUsStateRouter(r)

			// Register Comment Type Routes
			registerCommentTypeRouter(r)

			// Register Delay Code Routes
			registerDelayCodeRouter(r)

			// Register Fleet Code Routes
			registerFleetCodeRouter(r)

			// Register Location Category Routes
			registerLocationCategoryRouter(r)

			// Register Equipment Type Routes
			registerEquipmentTypeRouter(r)

			// Register Equipment Manufacturer Routes
			registerEquipmentManufacturerRouter(r)

			// Register Shipment Type Routes
			registerShipmentTypeRouter(r)

			// Register Service Type Routes
			registerServiceTypeRouter(r)

			// Register Qualifier Code Routes
			registerQualifierCodeRouter(r)

			// Register Table Change Alert Routes
			registerTableChangeAlertRouter(r)
		})
	})

	return r
}
