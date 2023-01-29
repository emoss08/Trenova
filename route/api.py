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
from rest_framework import permissions

from route import models, serializers
from utils.views import OrganizationMixin


class RouteViewSet(OrganizationMixin):
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
        "origin",
        "destination",
    )


class RouteControlViewSet(OrganizationMixin):
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
