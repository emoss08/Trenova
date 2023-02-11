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

from django.db.models import QuerySet
from drf_spectacular.types import OpenApiTypes
from drf_spectacular.utils import OpenApiParameter, extend_schema
from rest_framework import permissions, viewsets
from rest_framework.decorators import api_view
from rest_framework.request import Request
from rest_framework.response import Response

from core.health_check.cache_backend import CacheBackendHealthCheck
from core.health_check.celery_backend import CeleryHealthCheck
from core.health_check.database_backend import DatabaseHealthCheck
from core.health_check.disk_backend import DiskUsageHealthCheck
from core.health_check.redis_backend import RedisHealthCheck
from core.health_check.storage_backend import FileStorageHealthCheck
from organization import models, serializers
from utils.views import OrganizationMixin


class OrganizationViewSet(viewsets.ModelViewSet):
    """
    A viewset for viewing and editing organization instances.

    The viewset provides default operations for creating, updating, and deleting organizations,
    as well as listing and retrieving organizations. It uses the `OrganizationSerializer`
    class to convert the organization instances to and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this viewset.
    Filtering is also available, with the ability to filter by organization ID, name, and
    description.
    """

    serializer_class = serializers.OrganizationSerializer
    queryset = models.Organization.objects.all()

    def get_queryset(self) -> QuerySet[models.Organization]:
        """Filter the queryset to only include the current user

        Returns:
            QuerySet[models.Organization]: Filtered queryset
        """

        return self.queryset.prefetch_related(
            "depots",
            "depots__details",
        )


@extend_schema(
    parameters=[
        OpenApiParameter(
            "organizations_pk", OpenApiTypes.UUID, OpenApiParameter.PATH, required=True
        ),
    ]
)
class DepotViewSet(OrganizationMixin):
    """
    Depot ViewSet to manage requests to the depot endpoint
    """

    serializer_class = serializers.DepotSerializer
    queryset = models.Depot.objects.all()


@extend_schema(
    parameters=[
        OpenApiParameter(
            "organizations_pk", OpenApiTypes.UUID, OpenApiParameter.PATH, required=True
        ),
    ]
)
class DepartmentViewSet(OrganizationMixin):
    """
    Department ViewSet to manage requests to the department endpoint
    """

    serializer_class = serializers.DepartmentSerializer
    queryset = models.Department.objects.all()


class EmailProfileViewSet(OrganizationMixin):
    """
    EmailProfile ViewSet to manage requests to the Email profile endpoint
    """

    serializer_class = serializers.EmailProfileSerializer
    queryset = models.EmailProfile.objects.all()


class EmailControlViewSet(OrganizationMixin):
    """
    EmailControl ViewSet to manage requests to the email control endpoint
    """

    queryset = models.EmailControl.objects.all()
    serializer_class = serializers.EmailControlSerializer
    permission_classes = [permissions.IsAdminUser]
    http_method_names = ["get", "put", "patch", "head", "options"]


class EmailLogViewSet(viewsets.ModelViewSet):
    """
    EmailLog ViewSet to manage requests to the email log endpoint
    """

    queryset = models.EmailLog.objects.all()
    serializer_class = serializers.EmailLogSerializer
    permission_classes = [permissions.IsAdminUser]
    http_method_names = ["get", "head", "options"]


@extend_schema(
    tags=["Health Check"],
    description="Returns the health status of various components of the system.",
    responses={
        (200, "application/json"): {
            "type": "object",
            "properties": {
                "cache_backend": {
                    "type": "array",
                    "items": {
                        "type": "object",
                        "properties": {
                            "name": {"type": "string"},
                            "status": {"type": "string"},
                            "time": {"type": "number"},
                        },
                    },
                },
                "storage": {
                    "type": "object",
                    "properties": {
                        "status": {"type": "string"},
                        "time": {"type": "number"},
                    },
                },
                "redis": {
                    "type": "object",
                    "properties": {
                        "status": {"type": "string"},
                        "time": {"type": "number"},
                    },
                },
                "disk_usage": {
                    "type": "object",
                    "properties": {
                        "status": {"type": "string"},
                        "total": {"type": "number"},
                        "used": {"type": "number"},
                        "free": {"type": "number"},
                        "time": {"type": "number"},
                    },
                },
                "celery": {
                    "type": "object",
                    "properties": {
                        "status": {"type": "string"},
                        "time": {"type": "number"},
                    },
                },
                "database": {
                    "type": "object",
                    "properties": {
                        "status": {"type": "string"},
                        "time": {"type": "number"},
                    },
                },
            },
        }
    },
)
@api_view(["GET"])
def health_check(request: Request) -> Response:
    """
    Health check endpoint that returns the health status of various components of the system.

    Returns:
        Response: A dictionary that contains the health status of the cache backend, storage backend, redis, disk usage, celery, and database.
    """

    health_status = {
        "cache_backend": CacheBackendHealthCheck.check_caches_and_time(),
        "storage": FileStorageHealthCheck.check_file_storage(),
        "redis": RedisHealthCheck.check_redis(),
        "disk_usage": DiskUsageHealthCheck().check_disk_usage_and_time(),
        "celery": CeleryHealthCheck.check_celery(),
        "database": DatabaseHealthCheck.check_database(),
    }

    return Response(health_status)


class TaxRateViewSet(OrganizationMixin):
    """
    TaxRate ViewSet to manage requests to the tax rate endpoint
    """

    serializer_class = serializers.TaxRateSerializer
    queryset = models.TaxRate.objects.all()
