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

from rest_framework import routers

from accounts import api as accounts_api
from organization import api as org_api

router = routers.DefaultRouter()

router.register(r"users", accounts_api.UserViewSet, basename="user")
router.register(r"job_title", accounts_api.JobTitleViewSet, basename="job_title")
router.register(r"organizations", org_api.OrgViewSet, basename="organization")
router.register(r"depots", org_api.DepotViewSet, basename="depot")
router.register(r"departments", org_api.DepartmentViewSet, basename="department")

urlpatterns = [
    path("__debug__/", include("debug_toolbar.urls")),
    path("admin/doc/", include("django.contrib.admindocs.urls")),
    path("admin/", admin.site.urls),
    path("api/", include(router.urls)),
    path("api/token/provision/", accounts_api.TokenProvisionView.as_view(), name="token"),
    path("api/token/verify/", accounts_api.TokenVerifyView.as_view(), name="token"),
]

urlpatterns += static(settings.MEDIA_URL, document_root=settings.MEDIA_ROOT)  # type: ignore
urlpatterns += static(settings.STATIC_URL, document_root=settings.STATIC_ROOT)  # type: ignore
urlpatterns += [path("silk/", include("silk.urls", namespace="silk"))]
