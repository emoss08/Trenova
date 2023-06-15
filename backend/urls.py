# --------------------------------------------------------------------------------------------------
#  COPYRIGHT(c) 2023 MONTA                                                                         -
#                                                                                                  -
#  This file is part of Monta.                                                                     -
#                                                                                                  -
#  The Monta software is licensed under the Business Source License 1.1. You are granted the right -
#  to copy, modify, and redistribute the software, but only for non-production use or with a total -
#  of less than three server instances. Starting from the Change Date (November 16, 2026), the     -
#  software will be made available under version 2 or later of the GNU General Public License.     -
#  If you use the software in violation of this license, your rights under the license will be     -
#  terminated automatically. The software is provided "as is," and the Licensor disclaims all      -
#  warranties and conditions. If you use this license's text or the "Business Source License" name -
#  and trademark, you must comply with the Licensor's covenants, which include specifying the      -
#  Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use     -
#  Grant, and not modifying the license in any other way.                                          -
# --------------------------------------------------------------------------------------------------

from django.conf import settings
from django.conf.urls.static import static
from django.contrib import admin
from django.urls import include, path
from drf_spectacular.views import (
    SpectacularAPIView,
    SpectacularRedocView,
    SpectacularSwaggerView,
)
from rest_framework_nested import routers

from accounting import api as accounting_api
from accounts import api as accounts_api
from billing import api as billing_api
from commodities import api as commodities_api
from customer import api as customer_api
from dispatch import api as dispatch_api
from equipment import api as equipment_api
from integration import api as integration_api
from invoicing import api as invoicing_api
from location import api as location_api
from movements import api as movement_api
from order import api as order_api
from organization import api as org_api
from plugin import api as plugin_api
from reports import api as reports_api
from reports import views as reports_views
from route import api as route_api
from stops import api as stops_api
from worker import api as worker_api

router = routers.DefaultRouter()

# Accounts Routing
router.register(r"users", accounts_api.UserViewSet, basename="users")
router.register(r"job_titles", accounts_api.JobTitleViewSet, basename="job-titles")
router.register(r"groups", accounts_api.GroupViewSet, basename="groups")
router.register(r"permissions", accounts_api.PermissionViewSet, basename="permissions")

# Accounting Routes
router.register(
    r"gl_accounts",
    accounting_api.GeneralLedgerAccountViewSet,
    basename="gl-accounts",
)
router.register(
    r"revenue_codes", accounting_api.RevenueCodeViewSet, basename="revenue-codes"
)
router.register(
    r"division_codes", accounting_api.DivisionCodeViewSet, basename="division-codes"
)

# Organization Routing
router.register(r"organizations", org_api.OrganizationViewSet, basename="organization")
organization_router = routers.NestedSimpleRouter(
    router, r"organizations", lookup="organizations"
)
# organization/<str:pk>/depots
organization_router.register(
    r"depots", org_api.DepotViewSet, basename="organization-depot"
)
# organization/<str:pk>/departments
router.register(r"departments", org_api.DepartmentViewSet, basename="departments")
router.register(r"email_control", org_api.EmailControlViewSet, basename="email-control")
router.register(
    r"email_profiles", org_api.EmailProfileViewSet, basename="email-profiles"
)
router.register(r"email_log", org_api.EmailLogViewSet, basename="email-log")
router.register(r"tax_rates", org_api.TaxRateViewSet, basename="tax-rates")
router.register(
    r"table_change_alerts",
    org_api.TableChangeAlertViewSet,
    basename="table-change-alerts",
)
router.register(
    r"notification_types",
    org_api.NotificationTypeViewSet,
    basename="notification-types",
)
router.register(
    r"notification_settings",
    org_api.NotificationSettingViewSet,
    basename="notification-settings",
)

# Worker Routing
router.register(r"workers", worker_api.WorkerViewSet, basename="worker")
router.register(
    r"worker_profiles", worker_api.WorkerProfileViewSet, basename="worker-profile"
)
router.register(
    r"worker_comments", worker_api.WorkerCommentViewSet, basename="worker-comment"
)
router.register(
    r"worker_contacts", worker_api.WorkerContactViewSet, basename="worker-contact"
)

# Billing Routing
router.register(
    r"billing_control", billing_api.BillingControlViewSet, basename="billing-control"
)
router.register(
    r"billing_queue", billing_api.BillingQueueViewSet, basename="billing-queue"
)
router.register(
    r"billing_history", billing_api.BillingHistoryViewSet, basename="billing-history"
)
router.register(r"charge_types", billing_api.ChargeTypeViewSet, basename="charge-type")
router.register(
    r"accessorial_charges",
    billing_api.AccessorialChargeViewSet,
    basename="accessorial-charges",
)
router.register(
    r"document_classifications",
    billing_api.DocumentClassificationViewSet,
    basename="document-classifications",
)
router.register(
    r"billing_transfer_logs",
    billing_api.BillingTransferLogViewSet,
    basename="billing-transfer-logs",
)

# Commodity Routing
router.register(
    r"hazardous_materials",
    commodities_api.HazardousMaterialViewSet,
    basename="hazardous-materials",
)
router.register(r"commodities", commodities_api.CommodityViewSet, basename="commodity")

# Customer Routing
router.register(r"customers", customer_api.CustomerViewSet, basename="customer")
router.register(
    r"customer_fuel_tables",
    customer_api.CustomerFuelTableViewSet,
    basename="customer-fuel-tables",
)
router.register(
    r"customer_rule_profiles",
    customer_api.CustomerRuleProfileViewSet,
    basename="customer-rule-profiles",
)
router.register(
    r"customer_billing_profiles",
    customer_api.CustomerBillingProfileViewSet,
    basename="customer-billing-profiles",
)

# Equipment Routing
router.register(
    r"equipment_types", equipment_api.EquipmentTypeViewSet, basename="equipment-types"
)
router.register(r"tractors", equipment_api.TractorViewSet, basename="tractor")
router.register(r"trailers", equipment_api.TrailerViewSet, basename="trailer")
router.register(
    r"equipment_manufacturers",
    equipment_api.EquipmentManufacturerViewSet,
    basename="equipment-manufacturers",
)
router.register(
    r"equipment_maintenance_plans",
    equipment_api.EquipmentMaintenancePlanViewSet,
    basename="equipment-maintenance-plans",
)
# Location Routing
router.register(
    r"locations_categories",
    location_api.LocationCategoryViewSet,
    basename="location-categories",
)
router.register(r"locations", location_api.LocationViewSet, basename="locations")
router.register(
    r"location_contacts",
    location_api.LocationContactViewSet,
    basename="location-contacts",
)
router.register(
    r"location_comments",
    location_api.LocationCommentViewSet,
    basename="location-comments",
)

# Dispatch Routing
router.register(
    r"comment_types", dispatch_api.CommentTypeViewSet, basename="comment-types"
)
router.register(r"delay_codes", dispatch_api.DelayCodeViewSet, basename="delay-codes")
router.register(r"fleet_codes", dispatch_api.FleetCodeViewSet, basename="fleet-codes")
router.register(
    r"dispatch_control",
    dispatch_api.DispatchControlViewSet,
    basename="dispatch-control",
)
router.register(r"rates", dispatch_api.RateViewSet, basename="rates")
router.register(
    r"rate_billing_tables",
    dispatch_api.RateBillingTableViewSet,
    basename="rate-billing-tables",
)

# Integration Routing
router.register(
    r"integration_vendors",
    integration_api.IntegrationVendorViewSet,
    basename="integration-vendors",
)
router.register(
    r"integrations", integration_api.IntegrationViewSet, basename="integrations"
)
router.register(r"google_api", integration_api.GoogleAPIViewSet, basename="google-api")

# Route Routing
router.register(r"routes", route_api.RouteViewSet, basename="routes")
router.register(
    r"route_control", route_api.RouteControlViewSet, basename="route-control"
)

# Stops Routing
router.register(
    r"qualifier_codes", stops_api.QualifierCodeViewSet, basename="qualifier-codes"
)
router.register(
    r"stop_comments", stops_api.StopCommentViewSet, basename="stop-comments"
)
router.register(
    r"service_incidents", stops_api.ServiceIncidentViewSet, basename="service-incidents"
)
router.register(r"stops", stops_api.StopViewSet, basename="stops")

# Order Routing
router.register(
    r"order_control", order_api.OrderControlViewSet, basename="order-control"
)
router.register(r"order_types", order_api.OrderTypeViewSet, basename="order-types")
router.register(r"reason_codes", order_api.ReasonCodeViewSet, basename="reason-codes")
router.register(r"orders", order_api.OrderViewSet, basename="orders")
router.register(
    r"order_documents", order_api.OrderDocumentationViewSet, basename="order-documents"
)
router.register(
    r"order_comments", order_api.OrderCommentViewSet, basename="order-comments"
)
router.register(
    r"additional_charges",
    order_api.AdditionalChargeViewSet,
    basename="additional-charges",
)

# Movement Routing
router.register(r"movements", movement_api.MovementViewSet, basename="movements")

# Invoicing Routing
router.register(
    r"invoice_control", invoicing_api.InvoiceControlViewSet, basename="invoice_control"
)

# Reports Routing
router.register(
    r"custom_reports", reports_api.CustomReportViewSet, basename="custom_reports"
)

urlpatterns = [
    path("__debug__/", include("debug_toolbar.urls")),
    path("admin/doc/", include("django.contrib.admindocs.urls")),
    path("admin/", admin.site.urls),
    path("api/", include(router.urls)),
    path("api/", include(organization_router.urls)),
    path("api/schema/", SpectacularAPIView.as_view(), name="schema"),
    path(
        "api/docs/",
        SpectacularSwaggerView.as_view(url_name="schema"),
        name="swagger-ui",
    ),
    path(
        "api/change_password/",
        accounts_api.UpdatePasswordView.as_view(),
        name="change-password",
    ),
    path(
        "api/reset_password/",
        accounts_api.ResetPasswordView.as_view(),
        name="reset-password",
    ),
    path(
        "api/change_email/",
        accounts_api.UpdateEmailView.as_view(),
        name="change-email",
    ),
    path(
        "api/schema/redoc/",
        SpectacularRedocView.as_view(url_name="schema"),
        name="redoc",
    ),
    path(
        "api/login/",
        accounts_api.TokenProvisionView.as_view(),
        name="provision-token",
    ),
    path(
        "api/logout/",
        accounts_api.UserLogoutView.as_view(),
        name="logout-user",
    ),
    path(
        "api/verify_token/",
        accounts_api.TokenVerifyView.as_view(),
        name="verify-token",
    ),
    path("api/system_health/", org_api.health_check, name="system-health"),
    path("api/bill_invoice/", billing_api.bill_invoice_view, name="bill-order"),
    path("api/active_triggers/", org_api.active_triggers, name="active-triggers"),
    path("api/mass_bill_orders/", billing_api.mass_order_bill, name="bill-order"),
    path("api/active_sessions/", org_api.active_sessions, name="active-sessions"),
    path("api/active_threads/", org_api.active_threads, name="active-threads"),
    path(
        "api/table_columns/",
        reports_api.TableColumnsAPIView.as_view(),
        name="table-columns",
    ),
    path(
        "api/transfer_to_billing/",
        billing_api.transfer_to_billing,
        name="transfer-to-billing",
    ),
    path(
        "generate_excel_report/<str:report_id>/",
        reports_views.generate_excel_report,
        name="generate-excel-report",
    ),
    path("api/plugin_list/", plugin_api.get_plugin_list_api, name="plugin-list"),
    path("api/plugin_install/", plugin_api.plugin_install_api, name="plugin-list"),
    path(
        "api/cache_manager/",
        org_api.CacheManagerView.as_view(),
        name="cache-manager",
    ),
    path(
        "api/cache_manager/<str:model_path>/",
        org_api.CacheManagerView.as_view(),
        name="cache-manager-clear",
    ),
    path(
        "api/untransfer_invoice/",
        billing_api.untransfer_orders,
        name="untransfer-invoice",
    ),
    path(
        "api/get_columns/",
        reports_api.get_model_columns_api,
        name="get-model-columns",
    ),
    path(
        "api/generate_report/",
        reports_api.generate_report_api,
        name="generate-report",
    ),
]

urlpatterns += static(settings.MEDIA_URL, document_root=settings.MEDIA_ROOT)
urlpatterns += static(settings.STATIC_URL, document_root=settings.STATIC_ROOT)
urlpatterns += [path("silk/", include("silk.urls", namespace="silk"))]

admin.site.site_header = "Monta TMS Administration"
admin.site.site_title = "Monta TMS Administration"
admin.site.index_title = "Monta TMS Administration"
admin.site.empty_value_display = "N/A"
