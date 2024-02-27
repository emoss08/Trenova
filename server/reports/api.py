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

import logging
from typing import TYPE_CHECKING

from auditlog.models import LogEntry
from django.apps import apps
from django.core.cache import cache
from kombu.exceptions import OperationalError
from notifications.models import Notification
from rest_framework import exceptions, generics, status, viewsets
from rest_framework.decorators import api_view
from rest_framework.request import Request
from rest_framework.response import Response

from core.permissions import CustomObjectPermissions
from reports import models, serializers, tasks
from reports.helpers import ALLOWED_MODELS
from reports.selectors import get_audit_logs_by_model_name

logger = logging.getLogger(__name__)

if TYPE_CHECKING:
    from django.db.models import QuerySet


class TableColumnsAPIView(generics.GenericAPIView):
    """A class-based view for retrieving column information for a specified database table.

    Attributes:
        serializer_class (serializers.TableColumnSerializer): The serializer class used to serialize the response.
        authentication_classes (list): A list of authentication classes to use for this view.
        permission_classes (list): A list of permission classes to use for this view.

    Methods:
        get(request: Request) -> Response:
            Retrieves the column information for a specified database table.
    """

    serializer_class = serializers.TableColumnSerializer
    authentication_classes = []
    permission_classes = []

    def get(self, request: Request) -> Response:
        """Retrieves the column information for a specified database table.

        Args:
            request (Request): The HTTP request object containing the table_name parameter.

        Returns:
            Response: The HTTP response object containing the column information for the specified table.
        """

        if not (table_name := request.GET.get("table_name", None)):
            return Response(
                {"error": "Table name not provided."},
                status=status.HTTP_400_BAD_REQUEST,
            )
        if model := next(
            (
                app_model
                for app_model in apps.get_models()
                if app_model._meta.db_table.lower() == table_name.lower()
            ),
            None,
        ):
            columns = [
                {
                    "name": field.name,
                    "verbose_name": field.verbose_name,  # type: ignore
                }
                for field in model._meta.get_fields()
                if hasattr(field, "column")
            ]
            return Response({"columns": columns}, status=status.HTTP_200_OK)
        else:
            return Response(
                {"error": "Table not found."}, status=status.HTTP_404_NOT_FOUND
            )


class CustomReportViewSet(viewsets.ModelViewSet):
    """A viewset for managing CustomReport objects, with filters for name and table.

    Attributes:
        queryset (QuerySet): The queryset used for retrieving CustomReport objects.
        serializer_class (serializers.CustomReportSerializer): The serializer class used for CustomReport objects.
        filterset_fields (tuple): A tuple containing the names of the fields that can be used to filter CustomReport objects.
    """

    queryset = models.CustomReport.objects.all()
    serializer_class = serializers.CustomReportSerializer
    filterset_fields = (
        "name",
        "table",
    )
    permission_classes = [CustomObjectPermissions]

    def get_queryset(self) -> "QuerySet[models.CustomReport]":
        """Returns the queryset for this viewset, filtered by the organization of the current user.

        Returns:
            QuerySet[models.CustomReport]: The queryset for this viewset, filtered by the organization
             of the current user.
        """
        queryset: "QuerySet[models.CustomReport]" = self.queryset.filter(
            organization_id=self.request.user.organization_id  # type: ignore
        ).only(
            "id",
            "table",
            "name",
            "organization_id",
        )
        return queryset


@api_view(["GET"])
def get_model_columns_api(request: Request) -> Response:
    """API endpoint that allows users to get the allowed fields (columns) for a given model.

    This function takes a GET request with 'model_name' as a query parameter. If 'model_name' is not provided,
    it responds with a 400 Bad Request error.

    The function checks the 'model_name' against an allowed list of models. If the model is not in the allowed
    list, it responds with a 400 Bad Request error.

    If all validations pass, it responds with a 200 OK status and a list of the allowed fields for the model.

    Args:
        request (Request): The GET request sent to the endpoint. It should include 'model_name' in the
        query parameters.

    Returns:
        Response: Django Rest Framework Response object. If the request was processed successfully, the response
        includes a list of the allowed fields for the given model and HTTP status 200 (OK). In case of error(s),
        it includes an error message and the corresponding HTTP error status.

    Raises:
        KeyError: If the provided 'model_name' is not in the allowed list of models.
    """

    model_name = request.query_params.get("model_name", None)

    if not model_name:
        return Response(
            {"error": "Model name not provided."}, status=status.HTTP_400_BAD_REQUEST
        )

    # Cache key unique to the model
    cache_key = f"allowed_fields_{model_name}"

    # Try to get allowed fields from the cache
    allowed_model = cache.get(cache_key)

    if not allowed_model:
        try:
            allowed_model = ALLOWED_MODELS[model_name]
            # Store in cache for future requests
            cache.set(cache_key, allowed_model, timeout=3600)  # Cache for 1 hour
        except KeyError:
            return Response(
                {"error": f"Not allowed to generate reports for model: {model_name}"},
                status=status.HTTP_400_BAD_REQUEST,
            )

    return Response(
        {"results": allowed_model["allowed_fields"]}, status=status.HTTP_200_OK
    )


@api_view(["POST"])
def generate_report_api(request: Request) -> Response:
    """API endpoint that allows users to generate reports for a given model.

    Args:
        request (Request): Request object containing the report request data.

    Returns:
        Response: Django Rest Framework Response object. If the request was processed successfully, the response
        includes a message indicating that the report generation job was created and a 202 Accepted status. In case
        of error(s), it includes an error message and the corresponding HTTP error status.
    """
    serializer = serializers.ReportRequestSerializer(data=request.data)

    if not serializer.is_valid():
        return Response(serializer.errors, status=status.HTTP_400_BAD_REQUEST)

    validated_data = serializer.validated_data

    model_name = validated_data.get("model_name", None)
    columns = validated_data.get("columns", [])
    file_format = validated_data.get("file_format", "csv")
    delivery_method = validated_data.get("delivery_method", "download")
    email_recipients = validated_data.get("email_recipients", None)

    try:
        allowed_model = ALLOWED_MODELS[model_name]
    except KeyError:
        logger.error(f"Disallowed model access attempt: {model_name}")
        return Response(
            {"error": "Not allowed to generate reports for this model"},
            status=status.HTTP_403_FORBIDDEN,
        )

    allowed_fields = [field["value"] for field in allowed_model["allowed_fields"]]

    if any(column not in allowed_fields for column in columns):
        return Response(
            {"error": "One or more columns are not allowed"},
            status=status.HTTP_400_BAD_REQUEST,
        )

    parsed_email_recipients = (
        [email.strip() for email in email_recipients] if email_recipients else []
    )

    try:
        tasks.generate_report_task.delay(
            model_name=model_name,
            columns=columns,
            file_format=file_format,
            user_id=request.user.id,
            delivery_method=delivery_method,
            email_recipients=(
                parsed_email_recipients if delivery_method == "email" else None
            ),
        )
    except (exceptions.ValidationError, OperationalError) as e:
        logger.error(f"Exception in task initiation: {str(e)}")
        return Response(
            {"error": "Failed to initiate report generation task"},
            status=status.HTTP_500_INTERNAL_SERVER_ERROR,
        )

    return Response(
        {
            "results": "Report Generation Job Created. We will notify you when the report is ready."
        },
        status=status.HTTP_200_OK,
    )


class UserReportViewSet(viewsets.ModelViewSet):
    """A viewset for managing UserReport objects, with filters for name and table.

    Attributes:
        queryset (QuerySet): The queryset used for retrieving UserReport objects.
        serializer_class (serializers.UserReportSerializer): The serializer class used for UserReport objects.
        filterset_fields (tuple): A tuple containing the names of the fields that can be used to filter UserReport
        objects.
    """

    queryset = models.UserReport.objects.all()
    serializer_class = serializers.UserReportSerializer
    filterset_fields = ("user_id",)
    permission_classes = [CustomObjectPermissions]

    def get_queryset(self) -> "QuerySet[models.UserReport]":
        """Returns the queryset for the viewset.

        Returns:
            QuerySet[models.UserReport]: The queryset for the viewset.
        """

        queryset: "QuerySet[models.UserReport]" = self.queryset.filter(
            organization_id=self.request.user.organization_id, user_id=self.request.user.id  # type: ignore
        ).only(
            "id",
            "organization",
            "user",
            "report",
            "created",
            "modified",
        )
        return queryset


@api_view(["GET"])
def get_user_notifications(request: Request) -> Response:
    """API endpoint that allows users to get their unread notifications.

    This function takes a GET request and retrieves all unread notifications for the authenticated user
    making the request. It then returns a count of unread notifications and a list of all unread notifications.

    Args:
        request (Request): The GET request sent to the endpoint. The request should be authenticated with a user.

    Returns:
        Response: Django Rest Framework Response object. The response includes a dictionary with two keys:
        'unread_count' which holds the count of unread notifications, and 'unread_list' which holds a list of
        all unread notifications for the user. The HTTP status code is 200 (OK).
    """

    mark_as_read = request.query_params.get("mark_as_read", False)

    if mark_as_read:
        Notification.objects.filter(
            recipient=request.user,
            unread=True,
        ).update(unread=False)

    # Get all unread notifications for the user
    all_list = Notification.objects.filter(
        recipient=request.user,
        unread=True,
    ).all()

    # Notifications are serialized to JSON
    notification_data = serializers.NotificationSerializer(all_list, many=True).data

    data = {
        "unread_count": request.user.notifications.unread().count(),  # type: ignore
        "unread_list": notification_data,
    }

    return Response(data, status=status.HTTP_200_OK)


class LogEntryViewSet(viewsets.ModelViewSet):
    queryset = LogEntry.objects.all()
    serializer_class = serializers.LogEntrySerializer
    http_method_names = ["get"]

    def get_queryset(self) -> "QuerySet[LogEntry]":
        """Returns the queryset for the viewset.

        Returns:
            QuerySet[models.UserReport]: The queryset for the viewset.
        """

        model_name = self.request.query_params.get("model_name", None)

        if not model_name:
            raise exceptions.ValidationError("Query parameter 'model_name' is required")

        return get_audit_logs_by_model_name(
            model_name=model_name,
            organization_id=self.request.user.organization_id,  # type: ignore
        )
