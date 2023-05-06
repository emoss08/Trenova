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

from stops import models, serializers
from utils.views import OrganizationMixin


class QualifierCodeViewSet(OrganizationMixin):
    """A viewset for viewing and editing QualifierCode information in the system.

    The viewset provides default operations for creating, updating and deleting routes,
    as well as listing and retrieving routes. It uses `QualifierCodeSerializer` class to
    convert the route instances and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this viewset.

    Attributes:
        queryset (models.QualifierCode): The queryset of the viewset.
        serializer_class (serializers.QualifierCodeSerializer): The serializer class of the viewset.
    """

    queryset = models.QualifierCode.objects.all()
    serializer_class = serializers.QualifierCodeSerializer

    def get_queryset(self) -> QuerySet[models.QualifierCode]:
        queryset = self.queryset.filter(
            organization=self.request.user.organization  # type: ignore
        ).only(
            "organization_id",
            "code",
            "description",
        )
        return queryset


class StopCommentViewSet(OrganizationMixin):
    """A viewset for viewing and editing StopComment information in the system.

    The viewset provides default operations for creating, updating and deleting stop comments,
    as well as listing and retrieving stop comments. It uses `StopCommentSerializer` class to
    convert the stop comment instances and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this viewset.

    Attributes:
        queryset (models.StopComment): The queryset of the viewset.
        serializer_class (serializers.StopCommentSerializer): The serializer class of the viewset.
    """

    queryset = models.StopComment.objects.all()
    serializer_class = serializers.StopCommentSerializer

    def get_queryset(self) -> QuerySet[models.StopComment]:
        queryset = self.queryset.filter(
            organization=self.request.user.organization  # type: ignore
        ).only(
            "id",
            "stop_id",
            "comment_type_id",
            "qualifier_code_id",
            "comment",
            "entered_by_id",
        )
        return queryset


class StopViewSet(OrganizationMixin):
    """A viewset for viewing and editing Stop information in the system.

    The viewset provides default operations for creating, updating and deleting stops,
    as well as listing and retrieving stops. It uses `StopSerializer` class to
    convert the stop instances and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this viewset.

    Attributes:
        queryset (models.Stop): The queryset of the viewset.
        serializer_class (serializers.StopSerializer): The serializer class of the viewset.
    """

    queryset = models.Stop.objects.all()
    serializer_class = serializers.StopSerializer
    search_fields = (
        "status",
        "stop_type",
        "movement__ref_num",
        "location__code",
    )
    filterset_fields = ("status", "stop_type")
    ordering_fields = "__all__"

    def get_queryset(self) -> QuerySet[models.Stop]:
        queryset = self.queryset.filter(
            organization=self.request.user.organization  # type: ignore
        ).only(
            "id",
            "status",
            "sequence",
            "movement_id",
            "location_id",
            "pieces",
            "weight",
            "address_line",
            "organization_id",
            "appointment_time_window_start",
            "appointment_time_window_end",
            "arrival_time",
            "departure_time",
            "stop_type",
        )
        return queryset


class ServiceIncidentViewSet(OrganizationMixin):
    """A viewset for viewing and editing ServiceIncident information in the system.

    The viewset provides default operations for creating, updating and deleting service incidents,
    as well as listing and retrieving service incidents. It uses `ServiceIncidentSerializer` class to
    convert the service incident instances and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this viewset.
    Filter is also available, with the ability to filter by delay_code.

    Attributes:
        queryset (models.ServiceIncident): The queryset of the viewset.
        serializer_class (serializers.ServiceIncidentSerializer): The serializer class of the viewset.
    """

    queryset = models.ServiceIncident.objects.all()
    serializer_class = serializers.ServiceIncidentSerializer
    filterset_fields = ("delay_code",)
