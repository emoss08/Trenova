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
from rest_framework import permissions, viewsets

from route import models, serializers


class RouteViewSet(viewsets.ModelViewSet):
    """A viewset for viewing and editing Route information in the system.

    The viewset provides default operations for creating, updating and deleting routes,
    as well as listing and retrieving routes. It uses `RouteSerializer` class to
    convert the route instances and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this viewset.
    Filter is also available, with the ability to filter by id, origin and destination.
    """

    queryset = models.Route.objects.all()
    serializer_class = serializers.RouteSerializer
    filterset_fields = (
        "origin_location",
        "destination_location",
    )

    def get_queryset(self) -> QuerySet[models.Route]:
        queryset = self.queryset.filter(
            organization=self.request.user.organization  # type: ignore
        ).only(
            "id",
            "origin_location_id",
            "destination_location_id",
            "total_mileage",
            "duration",
            "organization_id",
            "distance_method",
        )
        return queryset


class RouteControlViewSet(viewsets.ModelViewSet):
    """A viewset for viewing and editing RouteControl information in the system.

    The viewset provides default operations for creating, updating and deleting route
    controls, as well as listing and retrieving route controls. It uses `RouteControlSerializer`
    class to convert the route control instances and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this viewset.
    Filter is also available, with the ability to filter by id, route and control.
    """

    queryset = models.RouteControl.objects.all()
    serializer_class = serializers.RouteControlSerializer
    permission_classes = [permissions.IsAdminUser]
    http_method_names = ["get", "put", "patch", "head", "options"]

    def get_queryset(self) -> QuerySet[models.RouteControl]:
        queryset = self.queryset.filter(
            organization=self.request.user.organization  # type: ignore
        ).only(
            "id",
            "organization_id",
            "distance_method",
            "mileage_unit",
            "generate_routes",
        )
        return queryset
