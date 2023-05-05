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

from django.db.models import Prefetch, QuerySet
from rest_framework import permissions

from movements.models import Movement
from order import models, serializers
from utils.views import OrganizationMixin


class OrderControlViewSet(OrganizationMixin):
    """A viewset for viewing and editing OrderControl in the system.

    The viewset provides default operations for creating, updating Order Control,
    as well as listing and retrieving Order Control. It uses the ``OrderControlSerializer``
    class to convert the order control instance to and from JSON-formatted data.

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
    http_method_names = ["get", "put", "patch", "head", "options"]


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
    filterset_fields = ("is_active",)


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

    def get_queryset(self) -> QuerySet[models.Order]:
        """Get the queryset for the viewset.

        The queryset is filtered by the organization of the user making the request.

        Returns:
            The filtered queryset.
        """
        return self.queryset.filter(
            organization=self.request.user.organization  # type: ignore
        ).prefetch_related(
            Prefetch(
                "movements",
                queryset=Movement.objects.filter(
                    organization=self.request.user.organization  # type: ignore
                ).only("id", "order_id"),
            )
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
        "accessorial_charge",
        "entered_by",
    )
