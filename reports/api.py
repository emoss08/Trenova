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
from typing import TYPE_CHECKING

from django.apps import apps
from notifications.helpers import get_notification_list
from rest_framework import exceptions, generics, status, viewsets
from rest_framework.decorators import api_view
from rest_framework.request import Request
from rest_framework.response import Response

from reports import models, serializers, tasks
from reports.exceptions import DisallowedModelException
from reports.helpers import ALLOWED_MODELS

if TYPE_CHECKING:
    from django.db.models import QuerySet


class TableColumnsAPIView(generics.GenericAPIView):
    """
    A class-based view for retrieving column information for a specified database table.

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
        """
        Retrieves the column information for a specified database table.

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

    def get_queryset(self) -> "QuerySet[models.CustomReport]":
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
    model_name = request.query_params.get("model_name", None)

    if not model_name:
        return Response(
            {"error": "Model name not provided."}, status=status.HTTP_400_BAD_REQUEST
        )

    try:
        allowed_model = ALLOWED_MODELS[model_name]
    except KeyError:
        return Response(
            {"error": f"Not allowed to generate reports for model: {model_name}"},
            status=400,
        )

    # No need to handle related fields separately anymore
    return Response(
        {"results": allowed_model["allowed_fields"]}, status=status.HTTP_200_OK
    )


@api_view(["POST"])
def generate_report_api(request: Request) -> Response:
    model_name = request.data.get("model_name", None)
    columns = request.data.get("columns", None)
    file_format = request.data.get("file_format", None)

    if not model_name:
        return Response(
            {"error": "Model name not provided."}, status=status.HTTP_400_BAD_REQUEST
        )
    if not columns:
        return Response(
            {"error": "Columns not provided."}, status=status.HTTP_400_BAD_REQUEST
        )
    if not file_format:
        return Response(
            {"error": "File format not provided."}, status=status.HTTP_400_BAD_REQUEST
        )

    try:
        allowed_model = ALLOWED_MODELS[model_name]
    except DisallowedModelException:
        return Response(
            {"error": f"Not allowed to generate reports for model: {model_name}"},
            status=status.HTTP_400_BAD_REQUEST,
        )

    # Check if columns are valid for the model
    for column in columns:
        if column not in allowed_model["allowed_fields"]:
            return Response(
                {"error": f"Invalid column for model: {column}"},
                status=status.HTTP_400_BAD_REQUEST,
            )

    try:
        tasks.generate_report_task.delay(
            model_name=model_name,
            columns=columns,
            file_format=file_format,
            user_id=request.user.id,
        )
        return Response(
            {
                "results": "Report Generation Job Created. We will notify you when the report is ready."
            },
            status=status.HTTP_202_ACCEPTED,
        )
    except exceptions.ValidationError as e:
        return Response({"error": str(e)}, status=status.HTTP_400_BAD_REQUEST)


class UserReportViewSet(viewsets.ModelViewSet):
    """A viewset for managing UserReport objects, with filters for name and table.

    Attributes:
        queryset (QuerySet): The queryset used for retrieving UserReport objects.
        serializer_class (serializers.UserReportSerializer): The serializer class used for UserReport objects.
        filterset_fields (tuple): A tuple containing the names of the fields that can be used to filter UserReport objects.
    """

    queryset = models.UserReport.objects.all()
    serializer_class = serializers.UserReportSerializer
    filterset_fields = ("user_id",)

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
    all_list = get_notification_list(request, "unread")

    data = {
        "unread_count": request.user.notifications.unread().count(),
        "unread_list": all_list,
    }

    return Response(data, status=status.HTTP_200_OK)
