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
import typing

from django.db.models import (
    Avg,
    DurationField,
    ExpressionWrapper,
    F,
    Prefetch,
    QuerySet,
)
from rest_framework import viewsets, response, status

from core.permissions import CustomObjectPermissions
from location import models, serializers

if typing.TYPE_CHECKING:
    from rest_framework.request import Request


class LocationCategoryViewSet(viewsets.ModelViewSet):
    """A viewset for viewing and editing location information in the system.

    The viewset provides default operations for creating, updating and deleting locations,
    as well as listing and retrieving locations. It uses `LocationSerializer`
    class to convert the location instances to and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this viewset.
    Filter is also available, with the ability to filter by Location ID, code and
    category.
    """

    queryset = models.LocationCategory.objects.all()
    serializer_class = serializers.LocationCategorySerializer
    permission_classes = [CustomObjectPermissions]


class LocationViewSet(viewsets.ModelViewSet):
    """A viewset for viewing and editing location information in the system.

    The viewset provides default operations for creating, updating and deleting locations,
    as well as listing and retrieving locations. It uses `LocationSerializer`
    class to convert the location instances to and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this viewset.
    Filter is also available, with the ability to filter by Location Category Name, Depot Name
    and is geocoded.
    """

    queryset = models.Location.objects.all()
    serializer_class = serializers.LocationSerializer
    filterset_fields = (
        "location_category__name",
        "depot__name",
        "is_geocoded",
        "status",
    )
    permission_classes = [CustomObjectPermissions]
    http_method_names = ["get", "post", "put", "patch", "head", "options"]

    def creat(
        self, request: "Request", *args: typing.Any, **kwargs: typing.Any
    ) -> response.Response:
        serializer = self.get_serializer(data=request.data)

        serializer.is_valid(raise_exception=True)
        self.perform_create(serializer)
        headers = self.get_success_headers(serializer.data)

        # Re-fetch the worker from the database to ensure all related data is fetched
        worker = models.Worker.objects.get(pk=serializer.instance.pk)  # type: ignore
        response_serializer = self.get_serializer(worker)

        return response.Response(
            response_serializer.data, status=status.HTTP_201_CREATED, headers=headers
        )

    def get_queryset(self) -> QuerySet[models.Location]:
        queryset = (
            self.queryset.filter(
                organization_id=self.request.user.organization_id  # type: ignore
            )
            .prefetch_related(
                Prefetch(
                    lookup="location_comments",
                    queryset=models.LocationComment.objects.filter(
                        organization_id=self.request.user.organization_id  # type: ignore
                    ).all(),
                ),
                Prefetch(
                    lookup="location_contacts",
                    queryset=models.LocationContact.objects.filter(
                        organization_id=self.request.user.organization_id  # type: ignore
                    ).all(),
                ),
            )
            .select_related("location_category")
            .annotate(
                wait_time_avg=Avg(
                    ExpressionWrapper(
                        F("stop__departure_time") - F("stop__arrival_time"),
                        output_field=DurationField(),
                    )
                )
            )
            .only(
                "organization_id",
                "business_unit_id",
                "id",
                "status",
                "code",
                "location_category_id",
                "location_category__color",
                "name",
                "depot_id",
                "description",
                "address_line_1",
                "address_line_2",
                "city",
                "state",
                "zip_code",
                "longitude",
                "latitude",
                "place_id",
                "is_geocoded",
                "created",
                "modified",
            )
        )
        return queryset


class LocationContactViewSet(viewsets.ModelViewSet):
    """A viewset for viewing and editing Location Contact information in the system.

    The viewset provides default operations for creating, updating and deleting Location
    Contacts, as well as listing and retrieving locations. It uses `LocationContactSerializer`
    class to convert the Location Contact instances to and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this viewset.
    """

    queryset = models.LocationContact.objects.all()
    serializer_class = serializers.LocationContactSerializer
    permission_classes = [CustomObjectPermissions]

    def get_queryset(self) -> QuerySet[models.LocationContact]:
        queryset = self.queryset.filter(
            organization_id=self.request.user.organization_id  # type: ignore
        ).only(
            "id",
            "location_id",
            "organization_id",
            "fax",
            "phone",
            "email",
            "name",
        )
        return queryset


class LocationCommentViewSet(viewsets.ModelViewSet):
    """A viewset for viewing and editing Location Comments information in the system.

    The viewset provides default operations for creating, updating and deleting Location
    Contacts, as well as listing and retrieving locations. It uses `LocationCommentSerializer`
    class to convert the Location Comments instances to and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this viewset.
    Filter is also available, with the ability to filter by Comment Type Name.
    """

    queryset = models.LocationComment.objects.all()
    serializer_class = serializers.LocationCommentSerializer
    permission_classes = [CustomObjectPermissions]

    def get_queryset(self) -> QuerySet[models.LocationComment]:
        queryset = self.queryset.filter().only(
            "id",
            "comment_type_id",
            "location_id",
            "entered_by_id",
            "comment",
            "organization_id",
        )
        return queryset
