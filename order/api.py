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

from utils.views import OrganizationViewSet
from order import models, serializers


class OrderControlViewSet(OrganizationViewSet):
    """A viewset for viewing and editing OrderControl in the system.

    The viewset provides default operations for creating, updating Order Control,
    as well as listing and retrieving Order Control. It uses the ``OrderControlSerializer`` class to
    convert the order control instance to and from JSON-formatted data.

    Attributes:
        queryset (QuerySet): A queryset of OrderControl objects that will be used to
        retrieve and update OrderControl objects.

        serializer_class (OrderControlSerializer): A serializer class that will be used to
        convert OrderControl objects to and from JSON-formatted data.

        permission_classes (list): A list of permission classes that will be used to
        determine if a user has permission to perform a particular action.
    """

    queryset = models.OrderControl.objects.all()
    permission_classes = [permissions.IsAdminUser]
    serializer_class = serializers.OrderControlSerializer


class OrderTypeViewSet(OrganizationViewSet):
    """A viewset for viewing and editing Order types in the system.

    The viewset provides default operations for creating, updating and deleting order types,
    as well as listing and retrieving Order Types. It uses the ``OrderTypesSerializer`` class to
    convert the order type instances to and from JSON-formatted data.

    Attributes:
        queryset (QuerySet): A queryset of OrderType objects that will be used to
        retrieve and update OrderType objects.

        serializer_class (OrderTypeSerializer): A serializer class that will be used to
        convert OrderType objects to and from JSON-formatted data.
    """

    queryset = models.OrderType.objects.all()
    serializer_class = serializers.OrderTypeSerializer


class ReasonCodeViewSet(OrganizationViewSet):
    """A viewset for viewing and editing Reason codes in the system.

    The viewset provides default operations for creating, updating and deleting reason codes,
    as well as listing and retrieving Reason Codes. It uses the ``ReasonCodeSerializer`` class to
    convert the reason code instances to and from JSON-formatted data.

    Attributes:
        queryset (QuerySet): A queryset of OrderType objects that will be used to
        retrieve and update OrderType objects.

        serializer_class (ReasonCodeSerializer): A serializer class that will be used to
        convert OrderType objects to and from JSON-formatted data.
    """

    queryset = models.ReasonCode.objects.all()
    serializer_class = serializers.ReasonCodeSerializer
