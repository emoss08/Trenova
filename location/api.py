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

from requests import Response
from rest_framework.request import Request

from location import models, selectors, serializers
from utils.views import OrganizationMixin


class LocationCategoryViewSet(OrganizationMixin):
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


class LocationViewSet(OrganizationMixin):
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

    def list(self, request: Request, *args: Any, **kwargs: Any) -> Response:  # type: ignore
        response = super().list(request, *args, **kwargs)

        locations = response.data["results"]

        for location in locations:
            location_obj = models.Location.objects.get(id=location["id"])
            wait_time_avg = selectors.get_avg_wait_time(location=location_obj)
            location["wait_time_avg"] = wait_time_avg

        return response  # type: ignore


class LocationContactViewSet(OrganizationMixin):
    """A viewset for viewing and editing Location Contact information in the system.

    The viewset provides default operations for creating, updating and deleting Location
    Contacts, as well as listing and retrieving locations. It uses `LocationContactSerializer`
    class to convert the Location Contact instances to and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this viewset.
    """

    queryset = models.LocationContact.objects.all()
    serializer_class = serializers.LocationContactSerializer


class LocationCommentViewSet(OrganizationMixin):
    """A viewset for viewing and editing Location Comments information in the system.

    The viewset provides default operations for creating, updating and deleting Location
    Contacts, as well as listing and retrieving locations. It uses `LocationCommentSerializer`
    class to convert the Location Comments instances to and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this viewset.
    Filter is also available, with the ability to filter by Comment Type Name.
    """

    queryset = models.LocationContact.objects.all()
    serializer_class = serializers.LocationCommentSerializer
