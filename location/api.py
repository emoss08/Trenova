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

from typing import Any

from django.db.models import Prefetch, QuerySet
from rest_framework import viewsets
from rest_framework.request import Request
from rest_framework.response import Response

from location import models, selectors, serializers


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
    filterset_fields = ("location_category__name", "depot__name", "is_geocoded")

    def list(self, request: Request, *args: Any, **kwargs: Any) -> Response:
        queryset = self.filter_queryset(self.get_queryset())

        # Annotate the queryset with average wait time
        queryset = selectors.get_avg_wait_time(queryset=queryset)

        page = self.paginate_queryset(queryset)
        if page is not None:
            serializer = self.get_serializer(page, many=True)
            data = serializer.data

            # Manually add wait_time_avg to the serialized data
            for item, obj in zip(data, page):
                item["wait_time_avg"] = obj.wait_time_avg

            return self.get_paginated_response(data)

        serializer = self.get_serializer(queryset, many=True)
        data = serializer.data

        # Manually add wait_time_avg to the serialized data
        for item, obj in zip(data, queryset):
            item["wait_time_avg"] = obj.wait_time_avg

        return Response(data)

    def get_queryset(self) -> QuerySet[models.Location]:
        queryset = (
            self.queryset.filter(
                organization=self.request.user.organization  # type: ignore
            )
            .prefetch_related(
                Prefetch(
                    lookup="location_comments",
                    queryset=models.LocationComment.objects.filter(
                        organization=self.request.user.organization  # type: ignore
                    ).only(
                        "id",
                    ),
                ),
                Prefetch(
                    lookup="location_contacts",
                    queryset=models.LocationContact.objects.filter(
                        organization=self.request.user.organization  # type: ignore
                    ).only(
                        "id",
                    ),
                ),
            )
            .only(
                "id",
                "organization_id",
                "description",
                "longitude",
                "address_line_1",
                "address_line_2",
                "is_geocoded",
                "zip_code",
                "latitude",
                "place_id",
                "city",
                "depot",
                "location_category_id",
                "code",
                "state",
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

    def get_queryset(self) -> QuerySet[models.LocationContact]:
        queryset = self.queryset.filter(
            organization=self.request.user.organization  # type: ignore
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
