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
from organization import api as org_api
from plugin import api as plugin_api
from reports import api as reports_api
from reports import views as reports_views
from route import api as route_api
from shipment import api as shipment_api
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
router.register(r"tags", accounting_api.TagViewSet, basename="tags")
router.register(
    r"finance_transactions", accounting_api.FinancialTransactionViewSet, basename="tags"
)
router.register(
    r"reconciliation_queue", accounting_api.ReconciliationQueueViewSet, basename="tags"
)
router.register(
    r"accounting_control", accounting_api.AccountingControlViewSet, basename="tags"
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
    r"billing_log_entry",
    billing_api.BillingLogEntryViewSet,
    basename="billing-log-entry",
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
    r"customer_email_profiles",
    customer_api.CustomerEmailProfileViewSet,
    basename="customer-email-profiles",
)
router.register(
    r"customer_rule_profiles",
    customer_api.CustomerRuleProfileViewSet,
    basename="customer-rule-profiles",
)
router.register(
    r"delivery_slots",
    customer_api.DeliverySlotViewSet,
    basename="delivery-slots",
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
    r"location_categories",
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
    r"feasibility_tool_control",
    dispatch_api.FeasibilityToolControlViewSet,
    basename="feasibility-tool-control",
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

# Shipment Routing
router.register(
    r"shipment_control",
    shipment_api.ShipmentControlViewSet,
    basename="shipment-control",
)
router.register(
    r"shipment_types", shipment_api.ShipmentTypeViewSet, basename="shipment-types"
)
router.register(
    r"reason_codes", shipment_api.ReasonCodeViewSet, basename="reason-codes"
)
router.register(r"shipments", shipment_api.ShipmentViewSet, basename="shipments")
router.register(
    r"shipment_documents",
    shipment_api.ShipmentDocumentationViewSet,
    basename="shipment-documents",
)
router.register(
    r"shipment_comments",
    shipment_api.ShipmentCommentViewSet,
    basename="shipment-comments",
)
router.register(
    r"additional_charges",
    shipment_api.AdditionalChargeViewSet,
    basename="additional-charges",
)
router.register(
    r"service_types",
    shipment_api.ServiceTypeViewSet,
    basename="service-types",
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
router.register(r"user_reports", reports_api.UserReportViewSet, basename="user_reports")
router.register(r"log_entries", reports_api.LogEntryViewSet, basename="log_entries")

urlpatterns = [
    path("admin/doc/", include("django.contrib.admindocs.urls")),
    path("admin/", admin.site.urls),
    path("api/", include(router.urls)),
    path("api/", include(organization_router.urls)),
    path("api/schema/", SpectacularAPIView.as_view(api_version="0.1.0"), name="schema"),
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
        "api/me/",
        accounts_api.UserDetailView.as_view(),
        name="me",
    ),
    path("api/system_health/", org_api.health_check, name="system-health"),
    path("api/bill_invoice/", billing_api.bill_invoice_view, name="bill-shipment"),
    path("api/active_triggers/", org_api.active_triggers, name="active-triggers"),
    path(
        "api/mass_bill_shipments/",
        billing_api.mass_shipments_bill,
        name="bill-shipment",
    ),
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
        billing_api.untransfer_shipment,
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
    path(
        "api/user/notifications/",
        reports_api.get_user_notifications,
        name="user-notifications",
    ),
    path(
        "api/billing/shipments_ready/",
        billing_api.get_shipments_ready,
        name="get-shipments-ready",
    ),
    path(
        "api/sessions/kick/",
        accounts_api.RemoveUserSessionView.as_view(),
        name="kick-user-session",
    ),
]

urlpatterns += static(settings.MEDIA_URL, document_root=settings.MEDIA_ROOT)
urlpatterns += static(settings.STATIC_URL, document_root=settings.STATIC_ROOT)

admin.site.site_header = "Monta TMS Administration"
admin.site.site_title = "Monta TMS Administration"
admin.site.index_title = "Monta TMS Administration"
admin.site.empty_value_display = "N/A"

if settings.DEBUG:
    urlpatterns += [path("silk/", include("silk.urls", namespace="silk"))]
