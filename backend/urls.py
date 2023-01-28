"""
COPYRIGHT 2022 MONTA

This file is part of Monta.

Monta is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

Monta is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with Monta.  If not, see <https://www.gnu.org/licenses/>.
"""

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
from location import api as location_api
from movements import api as movement_api
from order import api as order_api
from organization import api as org_api
from route import api as route_api
from stops import api as stops_api
from worker import api as worker_api

router = routers.DefaultRouter()

# Accounts Routing
router.register(r"users", accounts_api.UserViewSet, basename="users")
router.register(r"job_titles", accounts_api.JobTitleViewSet, basename="job-titles")

# Accounting Routes
router.register(
    r"gl_accounts",
    accounting_api.GeneralLedgerAccountViewSet,
    basename="general_ledger_accounts",
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
organization_router.register(
    r"departments", org_api.DepartmentViewSet, basename="organization-department"
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
router.register(r"equipment", equipment_api.EquipmentViewSet, basename="equipment")

# Location Routing
router.register(
    r"locations_categories",
    location_api.LocationCategoryViewSet,
    basename="location-categories",
)
router.register(r"locations", location_api.LocationViewSet, basename="locations")

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

# Integration Routing
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
        "api/schema/redoc/",
        SpectacularRedocView.as_view(url_name="schema"),
        name="redoc",
    ),
    path(
        "api/token/provision/",
        accounts_api.TokenProvisionView.as_view(),
        name="provision-token",
    ),
    path(
        "api/token/verify/", accounts_api.TokenVerifyView.as_view(), name="verify-token"
    ),
    path(
        "api/user/change_password/",
        accounts_api.UpdatePasswordView.as_view(),
        name="change-password",
    ),
]

urlpatterns += static(settings.MEDIA_URL, document_root=settings.MEDIA_ROOT)
urlpatterns += static(settings.STATIC_URL, document_root=settings.STATIC_ROOT)
urlpatterns += [path("silk/", include("silk.urls", namespace="silk"))]
