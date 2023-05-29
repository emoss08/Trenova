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
from rest_framework import permissions, viewsets

from movements.models import Movement
from order import models, serializers


class OrderControlViewSet(viewsets.ModelViewSet):
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

    def get_queryset(self) -> "QuerySet[models.OrderControl]":
        queryset = self.queryset.filter(
            organization=self.request.user.organization  # type: ignore
        ).only(
            "id",
            "organization_id",
            "auto_rate_orders",
            "calculate_distance",
            "enforce_rev_code",
            "enforce_voided_comm",
            "generate_routes",
            "enforce_commodity",
            "auto_sequence_stops",
            "auto_order_total",
            "enforce_origin_destination",
            "check_for_duplicate_bol",
            "remove_orders",
        )
        return queryset


class OrderTypeViewSet(viewsets.ModelViewSet):
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

    def get_queryset(self) -> "QuerySet[models.OrderType]":
        queryset = self.queryset.filter(
            organization=self.request.user.organization  # type: ignore
        ).only(
            "id",
            "organization_id",
            "name",
            "is_active",
            "description",
        )
        return queryset


class ReasonCodeViewSet(viewsets.ModelViewSet):
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

    def get_queryset(self) -> "QuerySet[models.ReasonCode]":
        queryset = self.queryset.filter(
            organization=self.request.user.organization  # type: ignore
        ).only(
            "id",
            "organization_id",
            "is_active",
            "code",
            "code_type",
            "description",
        )
        return queryset


class OrderViewSet(viewsets.ModelViewSet):
    queryset = models.Order.objects.all()
    serializer_class = serializers.OrderSerializer

    def get_queryset(self) -> "QuerySet[models.Order]":
        queryset = (
            self.queryset.filter(
                organization_id=self.request.user.organization_id  # type: ignore
            )
            .prefetch_related(
                Prefetch(
                    "additional_charges",
                    queryset=models.AdditionalCharge.objects.filter(
                        organization_id=self.request.user.organization_id  # type: ignore
                    )
                    .only("id", "order_id", "organization_id")
                    .all(),
                ),
                Prefetch(
                    lookup="movements",
                    queryset=Movement.objects.filter(
                        organization_id=self.request.user.organization_id  # type: ignore
                    )
                    .only("id", "order_id", "organization_id")
                    .all(),
                ),
                Prefetch(
                    lookup="order_documentation",
                    queryset=models.OrderDocumentation.objects.filter(
                        organization_id=self.request.user.organization_id  # type: ignore
                    )
                    .only("id", "order_id", "organization_id")
                    .all(),
                ),
                Prefetch(
                    lookup="order_comments",
                    queryset=models.OrderComment.objects.filter(
                        organization_id=self.request.user.organization_id  # type: ignore
                    )
                    .only("id", "order_id", "organization_id", "created")
                    .all(),
                ),
            )
            .only(
                "pro_number",
                "hazmat_id",
                "sub_total_currency",
                "id",
                "destination_address",
                "billing_transfer_date",
                "voided_comm",
                "destination_appointment_window_start",
                "weight",
                "billed",
                "sub_total",
                "bol_number",
                "other_charge_amount",
                "revenue_code_id",
                "temperature_min",
                "mileage",
                "other_charge_amount_currency",
                "auto_rate",
                "origin_appointment_window_start",
                "origin_appointment_window_end",
                "status",
                "freight_charge_amount",
                "freight_charge_amount_currency",
                "bill_date",
                "pieces",
                "destination_appointment_window_end",
                "entered_by_id",
                "consignee_ref_number",
                "origin_address",
                "origin_location_id",
                "equipment_type_id",
                "transferred_to_billing",
                "ready_to_bill",
                "order_type_id",
                "comment",
                "temperature_max",
                "destination_location_id",
                "commodity_id",
                "rate_method",
                "rate_id",
                "customer_id",
            )
        )

        return queryset


class OrderDocumentationViewSet(viewsets.ModelViewSet):
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

    def get_queryset(self) -> "QuerySet[models.OrderDocumentation]":
        queryset = self.queryset.filter(
            organization=self.request.user.organization  # type: ignore
        ).only(
            "id",
            "document",
            "order_id",
            "document_class_id",
        )
        return queryset


class OrderCommentViewSet(viewsets.ModelViewSet):
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

    def get_queryset(self) -> "QuerySet[models.OrderComment]":
        queryset = self.queryset.filter(
            organization=self.request.user.organization  # type: ignore
        ).only(
            "id",
            "comment",
            "order_id",
            "comment_type_id",
            "entered_by_id",
        )
        return queryset


class AdditionalChargeViewSet(viewsets.ModelViewSet):
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

    def get_queryset(self) -> "QuerySet[models.AdditionalCharge]":
        queryset = self.queryset.filter(
            organization=self.request.user.organization  # type: ignore
        ).only(
            "id",
            "accessorial_charge_id",
            "order_id",
            "description",
            "charge_amount",
            "unit",
            "entered_by_id",
            "sub_total",
            "charge_amount_currency",
            "sub_total_currency",
        )
        return queryset
