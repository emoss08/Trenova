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

from django.db.models import QuerySet
from drf_spectacular.types import OpenApiTypes
from drf_spectacular.utils import OpenApiParameter, extend_schema
from rest_framework import permissions, status, viewsets
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
from organization.selectors import get_active_triggers
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
    request=None,
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

    return Response(health_status, status=status.HTTP_200_OK)


@extend_schema(
    tags=["Active Triggers"],
    description="Get a list of current PostgreSQL triggers.",
    request=None,
    responses={
        (200, "application/json"): {
            "type": "array",
            "items": {
                "type": "object",
                "properties": {
                    "trigger_catalog": {"type": "string"},
                    "trigger_schema": {"type": "string"},
                    "trigger_name": {"type": "string"},
                    "event_manipulation": {"type": "string"},
                    "event_object_catalog": {"type": "string"},
                    "event_object_schema": {"type": "string"},
                    "event_object_table": {"type": "string"},
                    "action_order": {"type": "integer"},
                    "action_condition": {"type": "string"},
                    "action_statement": {"type": "string"},
                    "action_orientation": {"type": "string"},
                    "action_timing": {"type": "string"},
                },
            },
        },
    },
)
@api_view(["GET"])
def active_triggers(request: Request) -> Response:
    """
    View that retrieves a list of active triggers in the PostgreSQL database.

    Returns a response containing a list of dictionaries, where each dictionary
    represents a trigger and its metadata. The following fields are included in
    each dictionary: trigger_catalog, trigger_schema, trigger_name, event_manipulation,
    event_object_catalog, event_object_schema, event_object_table, action_order,
    action_condition, action_statement, action_orientation, and action_timing.

    Raises:
        NotImplementedError: If the database engine is not PostgreSQL.

    Returns:
        A Response object containing a list of dictionaries representing the triggers,
        with status code 200 (OK).
    """
    try:
        results = get_active_triggers()
        if results is None:
            return Response([], status=status.HTTP_200_OK)
        triggers = [
            {
                "trigger_catalog": result[0],
                "trigger_schema": result[1],
                "trigger_name": result[2],
                "event_manipulation": result[3],
                "event_object_catalog": result[4],
                "event_object_schema": result[5],
                "event_object_table": result[6],
                "action_order": result[7],
                "action_condition": result[8],
                "action_statement": result[9],
                "action_orientation": result[10],
                "action_timing": result[11],
            }
            for result in results
        ]
        return Response(triggers, status=status.HTTP_200_OK)
    except NotImplementedError as e:
        return Response(str(e), status=status.HTTP_400_BAD_REQUEST)


class TaxRateViewSet(OrganizationMixin):
    """
    TaxRate ViewSet to manage requests to the tax rate endpoint
    """

    serializer_class = serializers.TaxRateSerializer
    queryset = models.TaxRate.objects.all()


class TableChangeAlertViewSet(OrganizationMixin):
    """
    TableChangeAlert ViewSet to manage requests to the table change alert endpoint
    """

    serializer_class = serializers.TableChangeAlertSerializer
    queryset = models.TableChangeAlert.objects.all()
