# --------------------------------------------------------------------------------------------------
#  COPYRIGHT(c) 2024 Trenova                                                                       -
#                                                                                                  -
#  This file is part of Trenova.                                                                   -
#                                                                                                  -
#  The Trenova software is licensed under the Business Source License 1.1. You are granted the right
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
import json
import threading
from collections.abc import Sequence

import redis
from cacheops import invalidate_model
from django.apps import apps
from django.conf import settings
from django.core.exceptions import ValidationError
from django.db.models import Prefetch, QuerySet
from drf_spectacular.types import OpenApiTypes
from drf_spectacular.utils import OpenApiParameter, extend_schema
from rest_framework import permissions, status, views, viewsets
from rest_framework.decorators import api_view
from rest_framework.request import Request
from rest_framework.response import Response

from core import checks
from core.permissions import CustomObjectPermissions
from organization import exceptions, models, selectors, serializers


class OrganizationViewSet(viewsets.ModelViewSet):
    """A viewset for viewing and editing organization instances.

    The viewset provides default operations for creating, updating, and deleting organizations,
    as well as listing and retrieving organizations. It uses the `OrganizationSerializer`
    class to convert the organization instances to and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this viewset.
    Filtering is also available, with the ability to filter by organization ID, name, and
    description.
    """

    serializer_class = serializers.OrganizationSerializer
    queryset = models.Organization.objects.all()
    permission_classes = [permissions.IsAdminUser]

    def get_queryset(self) -> QuerySet[models.Organization]:
        """Filter the queryset to only include the current user

        Returns:
            QuerySet[models.Organization]: Filtered queryset
        """

        queryset = self.queryset.filter(
            id=self.request.user.organization_id  # type: ignore
        ).prefetch_related(  # type: ignore
            Prefetch(
                "depots",
                queryset=models.Depot.objects.filter(
                    organization_id=self.request.user.organization_id  # type: ignore
                ).only("id", "organization_id"),
            ),
        )
        return queryset


@extend_schema(
    parameters=[
        OpenApiParameter(
            "organizations_pk", OpenApiTypes.UUID, OpenApiParameter.PATH, required=True
        ),
    ]
)
class DepotViewSet(viewsets.ModelViewSet):
    """
    Depot ViewSet to manage requests to the depot endpoint
    """

    serializer_class = serializers.DepotSerializer
    queryset = models.Depot.objects.all()
    permission_classes = [CustomObjectPermissions]

    def get_queryset(self) -> QuerySet[models.Depot]:
        """The get_queryset function is used to filter the queryset of all depots
        to only those that belong to the organization associated with the user making
        the request. This function also selects related details and limits what fields are returned.

        Args:
            self: Access the attributes and methods of the class in python

        Returns:
            Queryset[models.Depot]: A queryset of depot objects
        """
        queryset: QuerySet[models.Depot] = self.queryset.filter(
            organization_id=self.request.user.organization_id  # type: ignore
        ).select_related("detail")
        return queryset


class DepartmentViewSet(viewsets.ModelViewSet):
    """
    Department ViewSet to manage requests to the department endpoint
    """

    serializer_class = serializers.DepartmentSerializer
    queryset = models.Department.objects.all()
    permission_classes = [CustomObjectPermissions]

    def get_queryset(self) -> QuerySet[models.Department]:
        """The get_queryset function is used to filter the queryset of departments
        to only those that belong to the organization of the user making this request.
        This is done by using a filter on organization_id, which will be set in our
        serializer's create function.

        Args:
            self: Access the attributes of the class

        Returns:
            QuerySet[models.Department]: A queryset
        """
        queryset: QuerySet[models.Department] = self.queryset.filter(
            organization_id=self.request.user.organization_id  # type: ignore
        )
        return queryset


class EmailProfileViewSet(viewsets.ModelViewSet):
    """
    EmailProfile ViewSet to manage requests to the Email profile endpoint
    """

    serializer_class = serializers.EmailProfileSerializer
    queryset = models.EmailProfile.objects.all()
    permission_classes = [CustomObjectPermissions]

    def get_queryset(self) -> QuerySet[models.EmailProfile]:
        """
        The get_queryset function is used to filter the queryset by organization_id.
        This is done so that users can only see email profiles for their own organization.
        The .only() function limits the fields returned in each object of the queryset, which helps with performance.

        Args:
            self: Refer to the class itself

        Returns:
            QuerySet[models.EmailProfile]: A queryset of emailprofile objects
        """
        queryset: QuerySet[models.EmailProfile] = self.queryset.filter(
            organization_id=self.request.user.organization_id  # type: ignore
        )
        return queryset


class EmailControlViewSet(viewsets.ModelViewSet):
    """
    EmailControl ViewSet to manage requests to the email control endpoint
    """

    queryset = models.EmailControl.objects.all()
    serializer_class = serializers.EmailControlSerializer
    permission_classes = [permissions.IsAdminUser]
    http_method_names = ["get", "put", "patch", "head", "options"]

    def get_queryset(self) -> QuerySet[models.EmailControl]:
        """The get_queryset function is used to filter the queryset by organization_id.
        This is done because we want to only return email controls that belong to
         the user's organization.


        Args:
            self: Access the class attributes and methods

        Returns:
            QuerySet[models.EmailControl]: A queryset of emailcontrol objects
        """
        queryset: QuerySet[models.EmailControl] = self.queryset.filter(
            organization_id=self.request.user.organization_id  # type: ignore
        ).only("id", "organization_id", "billing_email_profile_id")
        return queryset


class EmailLogViewSet(viewsets.ModelViewSet):
    """
    EmailLog ViewSet to manage requests to the email log endpoint
    """

    queryset = models.EmailLog.objects.all()
    serializer_class = serializers.EmailLogSerializer
    http_method_names = ["get", "head", "options"]
    permission_classes = [CustomObjectPermissions]


class TaxRateViewSet(viewsets.ModelViewSet):
    """
    TaxRate ViewSet to manage requests to the tax rate endpoint
    """

    serializer_class = serializers.TaxRateSerializer
    queryset = models.TaxRate.objects.all()
    permission_classes = [CustomObjectPermissions]

    def get_queryset(self) -> QuerySet[models.TaxRate]:
        """
        The get_queryset function is used to filter the queryset by organization_id.
        This is done so that users can only see tax rates for their own organization.
        The .only() function limits the fields returned in each object, which helps with performance.

        Args:
            self: Access the attributes and methods of the class

        Returns:
            A queryset[models.TaxRate]: A queryset of tax rate objects
        """
        queryset = self.queryset.filter(
            organization_id=self.request.user.organization_id  # type: ignore
        ).only(
            "id",
            "rate",
            "name",
        )
        return queryset


class TableChangeAlertViewSet(viewsets.ModelViewSet):
    """
    TableChangeAlert ViewSet to manage requests to the table change alert endpoint
    """

    serializer_class = serializers.TableChangeAlertSerializer
    queryset = models.TableChangeAlert.objects.all()
    permission_classes = [CustomObjectPermissions]

    def get_queryset(self) -> QuerySet[models.TableChangeAlert]:
        """The get_queryset function is used to filter the queryset based on the request.user's
         organization_id.This is done so that a user can only see alerts for
        their own organization, and not other organizations.

        Args:
            self: Refer to the class itself

        Returns:
            A queryset[models.TableChangeAlert]: A queryset of table change alert objects
        """

        queryset: QuerySet[models.TableChangeAlert] = self.queryset.filter(
            organization_id=self.request.user.organization_id  # type: ignore
        )
        return queryset


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
    """Health check endpoint that returns the health status of various components of the system.

    Returns:
        Response: A dictionary that contains the health status of the cache backend, storage backend, redis, disk usage,
        celery, and database.
    """
    health_status = {
        "cache_backend": checks.check_caches_and_time(),
        "storage": checks.check_file_storage(),
        "redis": checks.check_redis(),
        "disk_usage": checks.check_disk_usage_and_time(),
        "celery": checks.check_celery(),
        "database": checks.check_database(),
        "kafka": checks.check_kafka(),
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
    """View that retrieves a list of active triggers in the PostgreSQL database.

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
        results = selectors.get_active_triggers()
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


@api_view(["GET"])
def active_sessions(request: Request) -> Response:
    """View that retrieves a list of active sessions in the PostgreSQL database.

    Args:
        request (Request): The request object.

    Returns:
        Response: A Response object containing a list of dictionaries representing the sessions,
    """
    sessions = selectors.get_active_sessions()
    return Response(sessions, status=status.HTTP_200_OK)


@api_view(["GET"])
def active_threads(request: Request) -> Response:
    """View that retrieves a list of active threads in the PostgreSQL database.

    Args:
        request (Request): The request object.

    Returns:
        Response: A Response object containing a list of dictionaries representing the threads,
    """
    threads: list[threading.Thread] = threading.enumerate()
    thread_list = [
        {
            "name": thread.name,
            "ident": thread.ident,
            "daemon": thread.daemon,
            "is_alive": thread.is_alive(),
        }
        for thread in threads
    ]
    return Response(thread_list, status=status.HTTP_200_OK)


class CacheManagerView(views.APIView):
    """
    View that allows you to manage the cache.
    """

    permission_classes = [permissions.IsAdminUser]

    def get(self, request: Request) -> Response:
        """Get cache stats and keys

        Args:
            request (Request): The request object.

        Returns:
            Response: A Response object containing a list of dictionaries representing the cache keys,
        """
        search_term = request.query_params.get("search", "")

        # Parse Redis URL
        redis_url = settings.CACHEOPS_REDIS

        # Connect to Redis
        r = redis.StrictRedis.from_url(redis_url)

        # Get all Redis keys and filter based on your application's specific key patterns and the search term
        all_keys: Sequence[bytes] | list[bytes] = r.keys("*")
        cache_keys = [
            key.decode("utf-8")
            for key in all_keys
            if (key.startswith(b"schemes:") or key.startswith(b"conj:"))
            and search_term in key.decode("utf-8")
        ]

        # Cache stats
        info = r.info()
        cache_stats = {
            "hits": info["keyspace_hits"],
            "misses": info["keyspace_misses"],
            "total_keys": len(all_keys),
        }

        # Cache key size and content
        key_details = []
        for key in cache_keys:
            key_size = r.memory_usage(key)
            key_ttl = r.ttl(key)
            key_type = r.type(key).decode("utf-8")

            if key_type == "string":
                key_content = r.get(key)
                if key_content:
                    try:
                        key_content = json.loads(key_content)
                    except (json.JSONDecodeError, TypeError):
                        key_content = str(key_content)  # type: ignore
            elif key_type == "hash":
                key_content = r.hgetall(key)  # type: ignore
            elif key_type == "set":
                key_content = list(r.smembers(key))  # type: ignore
            else:
                key_content = None

            key_details.append(
                {
                    "key": key,
                    "size": key_size,
                    "ttl": key_ttl,
                    "type": key_type,
                    "content": key_content,
                }
            )

        return Response(
            {"cache_keys": key_details, "cache_stats": cache_stats},
            status=status.HTTP_200_OK,
        )

    def delete(self, request: Request, model_path: str | None = None) -> Response:
        """Clear cache for a specific model.

        Args:
            request (Request): The request object.
            model_path: The model path in the format of app_label.model_name.

        Returns:
            Response: A Response object containing a message indicating whether the cache was cleared successfully.
        """
        if not model_path:
            return Response(
                {"error": "Model path is required."}, status=status.HTTP_400_BAD_REQUEST
            )
        try:
            app_label, model_name = model_path.split(".")
            model = apps.get_model(app_label, model_name)
            invalidate_model(model)
            return Response(
                {"detail": f"Cache for {model_path} cleared."},
                status=status.HTTP_200_OK,
            )
        except exceptions.CacheOperationError as cache_exc:
            return Response(
                {"error": str(cache_exc)}, status=status.HTTP_400_BAD_REQUEST
            )


class NotificationTypeViewSet(viewsets.ModelViewSet):
    """
    API endpoint that allows notification types to be viewed or edited.
    """

    queryset = models.NotificationType.objects.all()
    serializer_class = serializers.NotificationTypeSerializer
    permission_classes = [permissions.IsAdminUser]

    def get_queryset(self) -> QuerySet[models.NotificationType]:
        """The get_queryset function is used to filter the queryset of NotificationType objects
        to only those that belong to the organization of the user making this request.
        This is done by using a QuerySet method called 'filter' which takes in a keyword argument,
        in this case 'organization_id', and returns all NotificationType objects where that field matches.
        The value for organization_id comes from self.request, which contains information about the
        current request being made.

        Args:
            self: Access the class attributes and methods

        Returns:
            QuerySet[models.NotificationType]: A queryset of notificationtype objects
        """
        queryset: QuerySet[models.NotificationType] = self.queryset.filter(
            organization_id=self.request.user.organization_id  # type: ignore
        ).only(
            "id",
            "organization_id",
            "name",
            "description",
        )
        return queryset


class NotificationSettingViewSet(viewsets.ModelViewSet):
    """
    API endpoint that allows notification settings to be viewed or edited.
    """

    queryset = models.NotificationSetting.objects.all()
    serializer_class = serializers.NotificationSettingSerializer
    permission_classes = [permissions.IsAdminUser]

    def get_queryset(self) -> QuerySet[models.NotificationSetting]:
        """The get_queryset function is used to filter the queryset of NotificationSettings
        to only those that belong to the organization of the user making this request.
        This is done by filtering on organization_id, which is a foreign key field in
        the NotificationSetting model. The only() function limits what fields are returned
        in order to reduce network traffic.

        Args:
            self: Refer to the class itself

        Returns:
            QuerySet[models.NotificationSetting]: A queryset of notificationsetting objects
        """

        queryset = models.NotificationSetting.objects.filter(
            organization_id=self.request.user.organization_id  # type: ignore
        ).only(
            "id",
            "organization_id",
            "notification_type_id",
            "send_notification",
            "email_recipients",
            "email_profile_id",
            "custom_content",
            "custom_subject",
        )
        return queryset


class OrganizationFeatureFlagView(views.APIView):
    """
    View that returns back the feature flags for the organization
    """

    permission_classes = [permissions.IsAuthenticated]

    def get(self, request: Request) -> Response:
        """Get the feature flags for the organization

        Args:
            request (Request): The request object.

        Returns:
            Response: A Response object containing a list of dictionaries representing the feature flags.
        """
        try:
            organization_id = request.user.organization_id  # type: ignore

            queryset = selectors.get_organization_feature_flags(
                organization_id=organization_id
            )

            serializer = serializers.OrganizationFeatureFlagSerializer(
                queryset, many=True, context={"request": request}
            )
            return Response(serializer.data, status=status.HTTP_200_OK)
        except AttributeError as e:
            print(e)
            return Response(
                {"detail": "Organization not found."}, status=status.HTTP_404_NOT_FOUND
            )


class UserOrganizationView(views.APIView):
    """
    View that returns back the organization for the user
    """

    permission_classes = [permissions.IsAuthenticated]

    def get(self, request: Request) -> Response:
        """Returns back the current organization for the user. This view is meant to just give back a single
        organization without having to use the OrganizationViewSet. That view will return all organizations business
        unit.

        Args:
            request (Request): The request object.

        Returns:
            Response: A Response object containing a dictionary representing the organization.
        """

        try:
            organization_id = request.user.organization_id  # type: ignore

            queryset = selectors.get_organization_by_id(organization_id=organization_id)

            serializer = serializers.OrganizationSerializer(
                queryset, context={"request": request}
            )
            return Response(serializer.data, status=status.HTTP_200_OK)
        except ValidationError:
            return Response(
                {"detail": "Organization not found."}, status=status.HTTP_404_NOT_FOUND
            )
