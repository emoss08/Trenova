"""
COPYRIGHT 2022 MONTA

This file is part of Monta.

Monta is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

Monta is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with Monta.  If not, see <https://www.gnu.org/licenses/>.
"""

from location import models, serializers
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
    Filter is also available, with the ability to filter by Location ID, code and
    category.
    """

    queryset = models.Location.objects.all()
    serializer_class = serializers.LocationSerializer
