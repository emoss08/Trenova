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


class ServiceIncidentViewSet(OrganizationViewSet):
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
