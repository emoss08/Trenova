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
from rest_framework import generics, viewsets
from rest_framework.request import Request
from rest_framework.response import Response

from reports import models, serializers

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
            return Response({"error": "Table name not provided."})
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
            return Response({"columns": columns})
        else:
            return Response({"error": "Table not found."})


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
            organization=self.request.user.organization  # type: ignore
        ).only(
            "id",
            "table",
            "name",
            "organization_id",
        )
        return queryset
