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
from core.permissions import CustomObjectPermissions
from dispatch import models, serializers
from django.db.models import Prefetch, QuerySet
from rest_framework import permissions, status, viewsets
from rest_framework.decorators import action
from rest_framework.request import Request
from rest_framework.response import Response


class CommentTypeViewSet(viewsets.ModelViewSet):
    """A viewset for viewing and editing Comment Type information in the system.

    The viewset provides default operations for creating, updating, and deleting Comment
    Types, as well as listing and retrieving Comment Types. It uses the `CommentTypeSerializer`
    class to convert the customer instances to and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this viewset.
    """

    queryset = models.CommentType.objects.all()
    serializer_class = serializers.CommentTypeSerializer
    permission_classes = [CustomObjectPermissions]
    filter_fields = ("status",)

    def get_queryset(self) -> QuerySet[models.CommentType]:
        queryset = self.queryset.filter(
            organization_id=self.request.user.organization_id  # type: ignore
        ).only(
            "id",
            "organization_id",
            "business_unit_id",
            "status",
            "name",
            "description",
            "created",
            "modified",
        )
        return queryset


class DelayCodeViewSet(viewsets.ModelViewSet):
    """A viewset for viewing and editing Delay Code information in the system.

    The viewset provides default operations for creating, updating, and deleting Delay
    Codes, as well as listing and retrieving Delay Codes. It uses the `DelayCodeSerializer`
    class to convert the customer instances to and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this viewset.
    """

    queryset = models.DelayCode.objects.all()
    serializer_class = serializers.DelayCodeSerializer
    permission_classes = [CustomObjectPermissions]

    def get_queryset(self) -> QuerySet[models.DelayCode]:
        queryset = self.queryset.filter(
            organization_id=self.request.user.organization_id  # type: ignore
        ).only(
            "organization_id",
            "status",
            "code",
            "description",
            "f_carrier_or_driver",
        )
        return queryset


class FleetCodeViewSet(viewsets.ModelViewSet):
    """A viewset for viewing and editing Fleet Code information in the system.

    The viewset provides default operations for creating, updating, and deleting Fleet Codes,
    as well as listing and retrieving Fleet Codes. It uses the `FleetCodeSerializer`
    class to convert the Fleet Codes instances to and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this viewset.
    Filtering is also available, with the ability to filter by is active.
    """

    queryset = models.FleetCode.objects.all()
    serializer_class = serializers.FleetCodeSerializer
    filterset_fields = ("status",)
    permission_classes = [CustomObjectPermissions]
    search_fields = ("code", "description", "status", "manager__username")

    def get_queryset(self) -> QuerySet[models.FleetCode]:
        queryset = self.queryset.filter(
            organization_id=self.request.user.organization_id  # type: ignore
        ).only(
            "business_unit_id",
            "organization_id",
            "code",
            "description",
            "status",
            "revenue_goal",
            "deadhead_goal",
            "manager_id",
            "mileage_goal",
            "modified",
            "created",
        )
        return queryset


class DispatchControlViewSet(viewsets.ModelViewSet):
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

    def get_queryset(self) -> QuerySet[models.DispatchControl]:
        queryset = self.queryset.filter(
            organization_id=self.request.user.organization_id  # type: ignore
        ).only(
            "organization_id",
            "record_service_incident",
            "trailer_continuity",
            "driver_time_away_restriction",
            "id",
            "grace_period",
            "dupe_trailer_check",
            "deadhead_target",
            "driver_assign",
            "prev_shipments_on_hold",
            "regulatory_check",
            "tractor_worker_fleet_constraint",
        )
        return queryset


class RateViewSet(viewsets.ModelViewSet):
    """A viewset for viewing and editing Rate information in the system.

    The viewset provides default operations for creating, updating, and deleting Rates,
    as well as listing and retrieving Rates. It uses the `RateSerializer`
    class to convert the Rate instances to and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this viewset.
    Filtering is also available, with the ability to filter by is active.
    """

    queryset = models.Rate.objects.all()
    serializer_class = serializers.RateSerializer
    permission_classes = [CustomObjectPermissions]
    filterset_fields = ("status",)
    search_fields = (
        "rate_number",
        "customer__code",
        "customer__name",
        "shipment_type__name",
        "equipment_type__name",
        "origin_location__code",
        "destination_location__code",
        "rate_method",
        "status",
    )

    @action(detail=False, methods=["get"])
    def get_new_rate_number(self, request: Request) -> Response:
        """Get the latest rate number

        Args:
            request(Request): Request objects

        Returns:
            Response: Response Object
        """
        new_rate_number = models.Rate.generate_rate_number()

        return Response({"rate_number": new_rate_number}, status=status.HTTP_200_OK)

    def get_queryset(self) -> QuerySet[models.Rate]:
        queryset = self.queryset.prefetch_related(
            Prefetch(
                "rate_billing_tables",
                queryset=models.RateBillingTable.objects.filter(
                    organization_id=self.request.user.organization_id  # type: ignore
                ).only(
                    "id",
                    "rate_id",
                    "accessorial_charge_id",
                    "description",
                    "unit",
                    "charge_amount",
                    "sub_total",
                ),
            )
        ).only(
            "id",
            "rate_number",
            "status",
            "customer_id",
            "effective_date",
            "expiration_date",
            "commodity_id",
            "shipment_type_id",
            "origin_location_id",
            "destination_location_id",
            "rate_method",
            "rate_amount",
            "equipment_type_id",
            "organization_id",
            "distance_override",
            "comments",
        )
        return queryset


class FeasibilityToolControlViewSet(viewsets.ModelViewSet):
    """A viewset for viewing and editing FeasibilityToolControl in the system.

    The viewset provides default operations for creating, updating Shipment Control,
    as well as listing and retrieving Feasibility Tool Control. It uses the ``FeasibilityToolControlSerializer``
    class to convert the Shipment Control instance to and from JSON-formatted data.

    Only admin users are allowed to access the views provided by this viewset.

    Attributes:
        queryset (QuerySet): A queryset of FeasibilityToolControl objects that will be used to
        retrieve and update ShipmentControl objects.

        serializer_class (FeasibilityToolControlSerializer): A serializer class that will be used to
        convert FeasibilityToolControl objects to and from JSON-formatted data.

        permission_classes (list): A list of permission classes that will be used to
        determine if a user has permission to perform a particular action.
    """

    queryset = models.FeasibilityToolControl.objects.all()
    permission_classes = [permissions.IsAdminUser]
    serializer_class = serializers.FeasibilityToolControlSerializer
    http_method_names = ["get", "put", "patch", "head", "options"]

    def get_queryset(self) -> "QuerySet[models.FeasibilityToolControl]":
        queryset = self.queryset.filter(
            organization_id=self.request.user.organization_id  # type: ignore
        ).only(
            "id",
            "organization_id",
            "business_unit_id",
            "mpw_operator",
            "mpw_criteria",
            "mpd_operator",
            "mpd_criteria",
            "mpg_operator",
            "mpg_criteria",
            "otp_operator",
            "otp_criteria",
        )
        return queryset
