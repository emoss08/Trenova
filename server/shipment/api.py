# --------------------------------------------------------------------------------------------------
#  COPYRIGHT(c) 2024 Trenova                                                                       -
#                                                                                                  -
#  This file is part of Trenova.                                                                   -
#                                                                                                  -
#  The Trenova software is licensed under the Business Source License 1.1. You are granted the right
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

from django.db.models import Count, Prefetch, Q, QuerySet
from rest_framework import permissions, status, viewsets
from rest_framework.decorators import action
from rest_framework.request import Request
from rest_framework.response import Response

from core.permissions import CustomObjectPermissions
from movements.models import Movement
from shipment import models, selectors, serializers
from stops.models import Stop
from utils.models import StatusChoices


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
    search_fields = ("status, code",)

    def get_queryset(self) -> "QuerySet[models.ShipmentType]":
        queryset = self.queryset.filter(
            organization_id=self.request.user.organization_id  # type: ignore
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
        )
        return queryset


class ShipmentViewSet(viewsets.ModelViewSet):
    """A viewset for viewing and editing shipments in the system.

    The viewset provides default operations for creating, updating and deleting shipments,
    as well as listing and retrieving shipments. It uses the ``ShipmentSerializer`` class to
    convert the shipment instances to and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this viewset.
    Filtering is also available, with the ability to filter by shipment type by pro_number, and customer.
    """

    queryset = models.Shipment.objects.all()
    serializer_class = serializers.ShipmentSerializer
    permission_classes = [CustomObjectPermissions]
    filterset_fields = (
        "pro_number",
        "customer",
        "status",
    )
    search_fields = ("pro_number", "customer__name", "bol_number")

    @action(detail=False, methods=["get"])
    def get_new_pro_number(self, request: Request) -> Response:
        """Get the latest pro number

        Args:
            request(Request): Request objects

        Returns:
            Response: Response Object
        """
        new_pro_number = selectors.next_pro_number(organization=request.user.organization_id)  # type: ignore

        return Response({"pro_number": new_pro_number}, status=status.HTTP_200_OK)

    @action(detail=False, methods=["POST"])
    def check_duplicate_bol(self, request: Request) -> Response:
        """Check for duplicate bol_number

        Args:
            request(Request): Request objects

        Returns:
            Response: Response Object
        """
        bol_number = request.data.get("bol_number")

        if models.Shipment.objects.filter(
            bol_number=bol_number,
            status__in=[StatusChoices.NEW, StatusChoices.IN_PROGRESS],
            organization_id=request.user.organization_id,  # type: ignore
        ).exists():
            return Response(
                {"valid": False, "message": "BOL Number already exists"},
                status=status.HTTP_200_OK,
            )

        return Response(status=status.HTTP_204_NO_CONTENT)

    @action(detail=False, methods=["get"])
    def get_shipment_count_by_status(self, request: Request) -> Response:
        """
        Get the total shipment count per status for the organization, with optional search filtering.

        Returns:
            Response: A response object containing the shipment count per status, along with the status representation.
        """

        if search_query := request.query_params.get("search"):
            search_conditions = (
                Q(pro_number__icontains=search_query)
                | Q(bol_number__icontains=search_query)
                | Q(customer__name__icontains=search_query)
            )
            filtered_queryset = self.queryset.filter(search_conditions)
        else:
            filtered_queryset = self.queryset

        shipment_count_by_status = (
            filtered_queryset.filter(
                organization_id=request.user.organization_id  # type: ignore
            )
            .values("status")
            .annotate(count=Count("status"))
            .order_by("status")
        )

        total_order_count = filtered_queryset.filter(
            organization_id=request.user.organization_id  # type: ignore
        ).count()

        return Response(
            {
                "results": shipment_count_by_status,
                "total_count": total_order_count,
            }
        )

    def get_queryset(self) -> "QuerySet[models.Shipment]":
        queryset = (
            self.queryset.filter(
                organization_id=self.request.user.organization_id  # type: ignore
            )
            .prefetch_related(
                Prefetch(
                    lookup="movements",
                    queryset=Movement.objects.filter(
                        organization_id=self.request.user.organization_id  # type: ignore
                    )
                    .prefetch_related(
                        Prefetch(
                            lookup="stops",
                            queryset=Stop.objects.filter(
                                organization_id=self.request.user.organization_id  # type: ignore
                            ).all(),
                        )
                    )
                    .all(),
                ),
                Prefetch(
                    lookup="shipment_documentation",
                    queryset=models.ShipmentDocumentation.objects.filter(
                        organization_id=self.request.user.organization_id  # type: ignore
                    ).all(),
                ),
                Prefetch(
                    lookup="shipment_comments",
                    queryset=models.ShipmentComment.objects.filter(
                        organization_id=self.request.user.organization_id  # type: ignore
                    ).all(),
                ),
            )
            .order_by("pro_number")
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
    search_fields = ("status, code",)

    def get_queryset(self) -> "QuerySet[models.ServiceType]":
        queryset = self.queryset.filter(
            organization_id=self.request.user.organization_id  # type: ignore
        )
        return queryset


class FormulaTemplateViewSet(viewsets.ModelViewSet):
    """A viewset for viewing and editing Formula Templates in the system.

    The viewset provides default operations for creating, updating and deleting Formula Templates,
    as well as listing and retrieving Formula Templates. It uses the ``FormulaTemplateSerializer``
    class to convert the Formula Template instances to and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this viewset.
    Filtering is also available, with the ability to filter by formula template by name, and status.

    Attributes:
        queryset (QuerySet): A queryset of FormulaTemplate objects that will be used to
        retrieve and update FormulaTemplate objects.

        serializer_class (FormulaTemplateSerializer): A serializer class that will be used to
        convert FormulaTemplate objects to and from JSON-formatted data.
    """

    queryset = models.FormulaTemplate.objects.all()
    serializer_class = serializers.FormulaTemplateSerializer
    filterset_fields = (
        "name",
        "shipment_type",
        "customer",
        "template_type",
    )
