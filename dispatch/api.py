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

from dispatch import models, serializers
from utils.views import OrganizationMixin


class CommentTypeViewSet(OrganizationMixin):
    """A viewset for viewing and editing Comment Type information in the system.

    The viewset provides default operations for creating, updating, and deleting Comment
    Types, as well as listing and retrieving Comment Types. It uses the `CommentTypeSerializer`
    class to convert the customer instances to and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this viewset.
    """

    queryset = models.CommentType.objects.all()
    serializer_class = serializers.CommentTypeSerializer


class DelayCodeViewSet(OrganizationMixin):
    """A viewset for viewing and editing Delay Code information in the system.

    The viewset provides default operations for creating, updating, and deleting Delay
    Codes, as well as listing and retrieving Delay Codes. It uses the `DelayCodeSerializer`
    class to convert the customer instances to and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this viewset.
    """

    queryset = models.DelayCode.objects.all()
    serializer_class = serializers.DelayCodeSerializer


class FleetCodeViewSet(OrganizationMixin):
    """A viewset for viewing and editing Fleet Code information in the system.

    The viewset provides default operations for creating, updating, and deleting Fleet Codes,
    as well as listing and retrieving Fleet Codes. It uses the `FleetCodeSerializer`
    class to convert the Fleet Codes instances to and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this viewset.
    Filtering is also available, with the ability to filter by is active.
    """

    queryset = models.FleetCode.objects.all()
    serializer_class = serializers.FleetCodeSerializer
    filterset_fields = ("is_active",)


class DispatchControlViewSet(OrganizationMixin):
    """A viewset for viewing and editing Dispatch Control in the system.

    The viewset provides default operations for updating, as well as listing and retrieving
    Dispatch Control. It uses the `DispatchControlSerializer` class to convert the Dispatch
    Control instances to and from JSON-formatted data.

    Only get, put, patch, head and options HTTP methods are allowed when using this viewset.
    Only Admin users are allowed to access the views provided by this viewset.
    """

    queryset = models.DispatchControl.objects.all()
    serializer_class = serializers.DispatchControlSerializer
    permission_classes = [permissions.IsAdminUser]
    http_method_names = ["get", "put", "patch", "head", "options"]


class RateViewSet(OrganizationMixin):
    """A viewset for viewing and editing Rate information in the system.

    The viewset provides default operations for creating, updating, and deleting Rates,
    as well as listing and retrieving Rates. It uses the `RateSerializer`
    class to convert the Rate instances to and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this viewset.
    Filtering is also available, with the ability to filter by is active.
    """

    queryset = models.Rate.objects.all()
    serializer_class = serializers.RateSerializer


class RateTableViewSet(OrganizationMixin):
    """
    Django Rest Framework ViewSet for the RateTable model.

    The RateTableViewSet class provides the CRUD operation for the RateTable model
    through the Django Rest Framework. The class is a subclass of OrganizationMixin,
    which provides the organization-related functionality.

    Attributes:
        queryset (models.RateTable.objects.all()): The default queryset for the viewset.
        serializer_class (serializers.RateTableSerializer): The serializer class for the viewset.
        filterset_fields (tuple): The fields to use for filtering the queryset.
    """

    queryset = models.RateTable.objects.all()
    serializer_class = serializers.RateTableSerializer
    filterset_fields = (
        "rate",
        "origin_location",
        "destination_location",
    )


class RateBillingTableViewSet(OrganizationMixin):
    """
    Django Rest Framework ViewSet for the RateBillingTable model.

    The RateBillingTableViewSet class provides the CRUD operation for the RateBillingTable model
    through the Django Rest Framework. The class is a subclass of OrganizationMixin,
    which provides the organization-related functionality.

    Attributes:
        queryset (models.RateBillingTable.objects.all()): The default queryset for the viewset.
        serializer_class (serializers.RateBillingTableSerializer): The serializer class for the viewset.
        filterset_fields (tuple): The fields to use for filtering the queryset.
    """

    queryset = models.RateBillingTable.objects.all()
    serializer_class = serializers.RateBillingTableSerializer
    filterset_fields = ("rate",)
