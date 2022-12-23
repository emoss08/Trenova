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
from rest_framework_nested import routers

from accounting import api as accounting_api
from accounts import api as accounts_api
from billing import api as billing_api
from commodities import api as commodities_api
from control_file import api as control_file_api
from customer import api as customer_api
from organization import api as org_api
from worker import api as worker_api
from equipment import api as equipment_api
from location import api as location_api
from dispatch import api as dispatch_api

router = routers.DefaultRouter()

# Accounts Routing
router.register(r"users", accounts_api.UserViewSet, basename="users")
router.register(r"job_titles", accounts_api.JobTitleViewSet, basename="job_titles")

# Accounting Routes
router.register(
    r"gl_accounts",
    accounting_api.GeneralLedgerAccountViewSet,
    basename="general_ledger_accounts",
)
router.register(
    r"revenue_codes", accounting_api.RevenueCodeViewSet, basename="revenue_codes"
)

# Organization Routing
router.register(r"organizations", org_api.OrganizationViewSet, basename="organizations")
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

# Billing Routing
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

# Control File Routing
router.register(
    r"google_api", control_file_api.GoogleAPIViewSet, basename="control_file"
)

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


urlpatterns = [
    path("__debug__/", include("debug_toolbar.urls")),
    path("admin/doc/", include("django.contrib.admindocs.urls")),
    path("admin/", admin.site.urls),
    path("api/", include(router.urls)),
    path("api/", include(organization_router.urls)),
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
