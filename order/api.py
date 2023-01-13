"""
COPYRIGHT(c) 2022 MONTA

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

from utils.views import OrganizationMixin
from order import models, serializers


class OrderControlViewSet(OrganizationMixin):
    """A viewset for viewing and editing OrderControl in the system.

    The viewset provides default operations for creating, updating Order Control,
    as well as listing and retrieving Order Control. It uses the ``OrderControlSerializer`` class to
    convert the order control instance to and from JSON-formatted data.

    Only admin users are allowed to access the views provided by this viewset.

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


class OrderTypeViewSet(OrganizationMixin):
    """A viewset for viewing and editing Order types in the system.

    The viewset provides default operations for creating, updating and deleting order types,
    as well as listing and retrieving Order Types. It uses the ``OrderTypesSerializer`` class to
    convert the order type instances to and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this viewset.
    Filtering is also available, with the ability to filter by order type by is_active.

    Attributes:
        queryset (QuerySet): A queryset of OrderType objects that will be used to
        retrieve and update OrderType objects.

        serializer_class (OrderTypeSerializer): A serializer class that will be used to
        convert OrderType objects to and from JSON-formatted data.
    """

    queryset = models.OrderType.objects.all()
    serializer_class = serializers.OrderTypeSerializer
    filterset_fields = ("is_active",)


class ReasonCodeViewSet(OrganizationMixin):
    """A viewset for viewing and editing Reason codes in the system.

    The viewset provides default operations for creating, updating and deleting reason codes,
    as well as listing and retrieving Reason Codes. It uses the ``ReasonCodeSerializer`` class to
    convert the reason code instances to and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this viewset.
    Filtering is also available, with the ability to filter by order type by is_active.

    Attributes:
        queryset (QuerySet): A queryset of OrderType objects that will be used to
        retrieve and update OrderType objects.

        serializer_class (ReasonCodeSerializer): A serializer class that will be used to
        convert OrderType objects to and from JSON-formatted data.
    """

    queryset = models.ReasonCode.objects.all()
    serializer_class = serializers.ReasonCodeSerializer


class OrderViewSet(OrganizationMixin):
    """A viewset for viewing and editing Orders in the system.

    The viewset provides default operations for creating, updating and deleting orders,
    as well as listing and retrieving Orders. It uses the ``OrderSerializer`` class to
    convert the order instances to and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this viewset.
    Filtering is also available, with the ability to filter by order type by order_type,
    revenue_code, customer, transferred_to_billing, equipment_type, commodity, entered_by
    and hazmat.


    Attributes:
        queryset (QuerySet): A queryset of Order objects that will be used to
        retrieve and update Order objects.

        serializer_class (OrderSerializer): A serializer class that will be used to
        convert Order objects to and from JSON-formatted data.
    """

    queryset = models.Order.objects.all()
    serializer_class = serializers.OrderSerializer
    filterset_fields = (
        "order_type",
        "revenue_code",
        "customer",
        "transferred_to_billing",
        "equipment_type",
        "commodity",
        "entered_by",
        "hazmat",
    )


class OrderDocumentationViewSet(OrganizationMixin):
    """A viewset for viewing and editing Order documentation in the system.

    The viewset provides default operations for creating, updating and deleting order documentation,
    as well as listing and retrieving Order Documentation. It uses the ``OrderDocumentationSerializer`` class to
    convert the order documentation instances to and from JSON-formatted data.

    Attributes:
        queryset (QuerySet): A queryset of OrderDocumentation objects that will be used to
        retrieve and update OrderDocumentation objects.

        serializer_class (OrderDocumentationSerializer): A serializer class that will be used to
        convert OrderDocumentation objects to and from JSON-formatted data.
    """

    queryset = models.OrderDocumentation.objects.all()
    serializer_class = serializers.OrderDocumentationSerializer


class OrderCommentViewSet(OrganizationMixin):
    """A viewset for viewing and editing Order comments in the system.

    The viewset provides default operations for creating, updating and deleting order comments,
    as well as listing and retrieving Order Comments. It uses the ``OrderCommentSerializer`` class to
    convert the order comment instances to and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this viewset.
    Filtering is also available, with the ability to filter by order type by order_type, and entered_by.

    Attributes:
        queryset (QuerySet): A queryset of OrderComment objects that will be used to
        retrieve and update OrderComment objects.

        serializer_class (OrderCommentSerializer): A serializer class that will be used to
        convert OrderComment objects to and from JSON-formatted data.
    """

    queryset = models.OrderComment.objects.all()
    serializer_class = serializers.OrderCommentSerializer
    filterset_fields = (
        "comment_type",
        "entered_by",
    )


class AdditionalChargeViewSet(OrganizationMixin):
    """A viewset for viewing and editing Additional charges in the system.

    The viewset provides default operations for creating, updating and deleting additional charges,
    as well as listing and retrieving Additional Charges. It uses the ``AdditionalChargeSerializer``
    class to convert the additional charge instances to and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this viewset.
    Filtering is also available, with the ability to filter by order type by charge, and entered_by.

    Attributes:
        queryset (QuerySet): A queryset of AdditionalCharge objects that will be used to
        retrieve and update AdditionalCharge objects.

        serializer_class (AdditionalChargeSerializer): A serializer class that will be used to
        convert AdditionalCharge objects to and from JSON-formatted data.
    """

    queryset = models.AdditionalCharge.objects.all()
    serializer_class = serializers.AdditionalChargeSerializer
    filterset_fields = (
        "charge",
        "entered_by",
    )
