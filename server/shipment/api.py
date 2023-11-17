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

from core.permissions import CustomObjectPermissions
from movements.models import Movement
from shipment import models, serializers


class ShipmentControlViewSet(viewsets.ModelViewSet):
    """A viewset for viewing and editing ShipmentControl in the system.

    The viewset provides default operations for creating, updating Shipment Control,
    as well as listing and retrieving Shipment Control. It uses the ``ShipmentControlSerializer``
    class to convert the Shipment Control instance to and from JSON-formatted data.

    Only admin users are allowed to access the views provided by this viewset.

    Attributes:
        queryset (QuerySet): A queryset of ShipmentControl objects that will be used to
        retrieve and update ShipmentControl objects.

        serializer_class (ShipmentControlSerializer): A serializer class that will be used to
        convert ShipmentControl objects to and from JSON-formatted data.

        permission_classes (list): A list of permission classes that will be used to
        determine if a user has permission to perform a particular action.
    """

    queryset = models.ShipmentControl.objects.all()
    permission_classes = [permissions.IsAdminUser]
    serializer_class = serializers.ShipmentControlSerializer
    http_method_names = ["get", "put", "patch", "head", "options"]

    def get_queryset(self) -> "QuerySet[models.ShipmentControl]":
        queryset = self.queryset.filter(
            organization_id=self.request.user.organization_id  # type: ignore
        ).only(
            "id",
            "organization_id",
            "auto_rate_shipment",
            "calculate_distance",
            "enforce_rev_code",
            "enforce_voided_comm",
            "generate_routes",
            "enforce_commodity",
            "auto_sequence_stops",
            "auto_shipment_total",
            "enforce_origin_destination",
            "check_for_duplicate_bol",
            "remove_shipment",
        )
        return queryset


class ShipmentTypeViewSet(viewsets.ModelViewSet):
    """A viewset for viewing and editing shipment types in the system.

    The viewset provides default operations for creating, updating and deleting shipment types,
    as well as listing and retrieving shipment types. It uses the ``ShipmentTypesSerializer`` class to
    convert the shipment type instances to and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this viewset.
    Filtering is also available, with the ability to filter by shipment type by is_active.

    Attributes:
        queryset (QuerySet): A queryset of ShipmentType objects that will be used to
        retrieve and update ShipmentType objects.

        serializer_class (ShipmentTypeSerializer): A serializer class that will be used to
        convert ShipmentType objects to and from JSON-formatted data.
    """

    queryset = models.ShipmentType.objects.all()
    serializer_class = serializers.ShipmentTypeSerializer
    filterset_fields = ("status",)
    permission_classes = [CustomObjectPermissions]

    def get_queryset(self) -> "QuerySet[models.ShipmentType]":
        queryset = self.queryset.filter(
            organization_id=self.request.user.organization_id  # type: ignore
        ).only(
            "id",
            "organization_id",
            "business_unit_id",
            "code",
            "status",
            "description",
            "created",
            "modified",
        )
        return queryset


class ReasonCodeViewSet(viewsets.ModelViewSet):
    """A viewset for viewing and editing Reason codes in the system.

    The viewset provides default operations for creating, updating and deleting reason codes,
    as well as listing and retrieving Reason Codes. It uses the ``ReasonCodeSerializer`` class to
    convert the reason code instances to and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this viewset.
    Filtering is also available, with the ability to filter by shipment type by is_active.

    Attributes:
        queryset (QuerySet): A queryset of ShipmentType objects that will be used to
        retrieve and update ShipmentType objects.
        serializer_class (ReasonCodeSerializer): A serializer class that will be used to
        convert ShipmentType objects to and from JSON-formatted data.
    """

    queryset = models.ReasonCode.objects.all()
    serializer_class = serializers.ReasonCodeSerializer
    filterset_fields = ("status",)
    permission_classes = [CustomObjectPermissions]

    def get_queryset(self) -> "QuerySet[models.ReasonCode]":
        queryset = self.queryset.filter(
            organization_id=self.request.user.organization_id  # type: ignore
        ).only(
            "id",
            "business_unit_id",
            "organization_id",
            "status",
            "code",
            "code_type",
            "description",
            "created",
            "modified",
        )
        return queryset


class ShipmentViewSet(viewsets.ModelViewSet):
    queryset = models.Shipment.objects.all()
    serializer_class = serializers.ShipmentSerializer
    permission_classes = [CustomObjectPermissions]
    filterset_fields = ("pro_number", "customer")

    def get_queryset(self) -> "QuerySet[models.Shipment]":
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
                    .only("id", "shipment_id", "organization_id")
                    .all(),
                ),
                Prefetch(
                    lookup="movements",
                    queryset=Movement.objects.filter(
                        organization_id=self.request.user.organization_id  # type: ignore
                    )
                    .only("id", "shipment_id", "organization_id")
                    .all(),
                ),
                Prefetch(
                    lookup="shipment_documentation",
                    queryset=models.ShipmentDocumentation.objects.filter(
                        organization_id=self.request.user.organization_id  # type: ignore
                    )
                    .only("id", "shipment_id", "organization_id")
                    .all(),
                ),
                Prefetch(
                    lookup="shipment_comments",
                    queryset=models.ShipmentComment.objects.filter(
                        organization_id=self.request.user.organization_id  # type: ignore
                    )
                    .only("id", "shipment_id", "organization_id", "created")
                    .all(),
                ),
            )
            .only(
                "pro_number",
                "hazmat_id",
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
                "auto_rate",
                "origin_appointment_window_start",
                "origin_appointment_window_end",
                "status",
                "freight_charge_amount",
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
                "shipment_type_id",
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


class ShipmentDocumentationViewSet(viewsets.ModelViewSet):
    """A viewset for viewing and editing shipment documentation in the system.

    The viewset provides default operations for creating, updating and deleting Shipment documentation,
    as well as listing and retrieving Shipment Documentation. It uses the ``ShipmentDocumentationSerializer`` class to
    convert the Shipment documentation instances to and from JSON-formatted data.

    Attributes:
        queryset (QuerySet): A queryset of ShipmentDocumentation objects that will be used to
        retrieve and update ShipmentDocumentation objects.

        serializer_class (ShipmentDocumentationSerializer): A serializer class that will be used to
        convert ShipmentDocumentation objects to and from JSON-formatted data.
    """

    queryset = models.ShipmentDocumentation.objects.all()
    serializer_class = serializers.ShipmentDocumentationSerializer
    permission_classes = [CustomObjectPermissions]

    def get_queryset(self) -> "QuerySet[models.ShipmentDocumentation]":
        queryset = self.queryset.filter(
            organization_id=self.request.user.organization_id  # type: ignore
        ).only(
            "id",
            "document",
            "shipment_id",
            "document_class_id",
        )
        return queryset


class ShipmentCommentViewSet(viewsets.ModelViewSet):
    """A viewset for viewing and editing Shipment comments in the system.

    The viewset provides default operations for creating, updating and deleting Shipment comments,
    as well as listing and retrieving Shipment Comments. It uses the ``ShipmentCommentSerializer`` class to
    convert the Shipment comment instances to and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this viewset.
    Filtering is also available, with the ability to filter by shipment type by shipment_type, and entered_by.

    Attributes:
        queryset (QuerySet): A queryset of ShipmentComment objects that will be used to
        retrieve and update ShipmentComment objects.

        serializer_class (ShipmentCommentSerializer): A serializer class that will be used to
        convert ShipmentComment objects to and from JSON-formatted data.
    """

    queryset = models.ShipmentComment.objects.all()
    serializer_class = serializers.ShipmentCommentSerializer
    filterset_fields = (
        "comment_type",
        "entered_by",
    )
    permission_classes = [CustomObjectPermissions]

    def get_queryset(self) -> "QuerySet[models.ShipmentComment]":
        queryset = self.queryset.filter(
            organization_id=self.request.user.organization_id  # type: ignore
        ).only(
            "id",
            "comment",
            "shipment_id",
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
    Filtering is also available, with the ability to filter by shipment type by charge, and entered_by.

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
    permission_classes = [CustomObjectPermissions]

    def get_queryset(self) -> "QuerySet[models.AdditionalCharge]":
        queryset = self.queryset.filter(
            organization_id=self.request.user.organization_id  # type: ignore
        ).only(
            "id",
            "accessorial_charge_id",
            "shipment_id",
            "description",
            "charge_amount",
            "unit",
            "entered_by_id",
            "sub_total",
        )
        return queryset


class ServiceTypeViewSet(viewsets.ModelViewSet):
    """A viewset for viewing and editing Service Type in the system.

    The viewset provides default operations for creating, updating and deleting service types,
    as well as listing and retrieving Service Types. It uses the ``ServiceTypeSerializer``
    class to convert the additional charge instances to and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this viewset.
    Filtering is also available, with the ability to filter by service type by code, and status.

    Attributes:
        queryset (QuerySet): A queryset of ServiceType objects that will be used to
        retrieve and update ServiceType objects.

        serializer_class (ServiceTypeSerializer): A serializer class that will be used to
        convert ServiceType objects to and from JSON-formatted data.
    """

    queryset = models.ServiceType.objects.all()
    serializer_class = serializers.ServiceTypeSerializer
    filterset_fields = ("status", "code")
    permission_classes = [CustomObjectPermissions]

    def get_queryset(self) -> "QuerySet[models.ServiceType]":
        queryset = self.queryset.filter(
            organization_id=self.request.user.organization_id  # type: ignore
        ).only(
            "id",
            "organization_id",
            "business_unit_id",
            "status",
            "code",
            "description",
            "created",
            "modified",
        )
        return queryset
