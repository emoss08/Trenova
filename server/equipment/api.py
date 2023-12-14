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
import typing

from django.db.models import Count, F, Prefetch, QuerySet
from rest_framework import response, status, viewsets

from core.permissions import CustomObjectPermissions
from equipment import models, serializers

if typing.TYPE_CHECKING:
    from rest_framework.request import Request


class EquipmentTypeViewSet(viewsets.ModelViewSet):
    """A viewset for viewing and editing customer information in the system.

    The viewset provides default operations for creating, updating, and deleting customers,
    as well as listing and retrieving customers. It uses the `CustomerSerializer`
    class to convert the customer instances to and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this viewset.
    Filtering is also available, with the ability to filter by customer ID, name, and
    code.
    """

    queryset = models.EquipmentType.objects.all()
    serializer_class = serializers.EquipmentTypeSerializer
    permission_classes = [CustomObjectPermissions]

    def create(
        self, request: "Request", *args: typing.Any, **kwargs: typing.Any
    ) -> response.Response:
        serializer = self.get_serializer(data=request.data)
        serializer.is_valid(raise_exception=True)
        self.perform_create(serializer)
        headers = self.get_success_headers(serializer.data)

        # Re-fetch the worker from the database to ensure all related data is fetched
        worker = models.EquipmentType.objects.get(pk=serializer.instance.pk)  # type: ignore
        response_serializer = self.get_serializer(worker)

        return response.Response(
            response_serializer.data, status=status.HTTP_201_CREATED, headers=headers
        )

    def get_queryset(self) -> QuerySet[models.EquipmentType]:
        queryset = self.queryset.filter(
            organization_id=self.request.user.organization_id  # type: ignore
        ).only(
            "organization_id",
            "business_unit_id",
            "id",
            "name",
            "description",
            "cost_per_mile",
            "equipment_class",
            "fixed_cost",
            "variable_cost",
            "height",
            "length",
            "width",
            "weight",
            "idling_fuel_usage",
            "exempt_from_tolls",
            "created",
            "modified",
        )
        return queryset


class TractorViewSet(viewsets.ModelViewSet):
    """A viewset for viewing and editing tractors information in the system.

    The viewset provides default operations for creating, updating, and deleting tractors,
    as well as listing and retrieving tractors. It uses the `TractorSerializer`
    class to convert the tractor instances to and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this viewset.
    Filtering is also available, with the ability to filter by is_active, manufacturer,
    has_berth, equipment_type__name, fleet__code and highway_use_tax.
    """

    queryset = models.Tractor.objects.all()
    serializer_class = serializers.TractorSerializer
    filterset_fields = (
        "status",
        "manufacturer",
        "equipment_type__name",
        "has_berth",
        "highway_use_tax",
    )
    permission_classes = [CustomObjectPermissions]

    def get_queryset(self) -> QuerySet[models.Tractor]:
        queryset = self.queryset.filter(
            organization_id=self.request.user.organization_id  # type: ignore
        ).all()
        return queryset


class TrailerViewSet(viewsets.ModelViewSet):
    """A viewset for viewing and editing customer information in the system.

    The viewset provides default operations for creating, updating, and deleting trailers,
    as well as listing and retrieving trailers. It uses the `Trailers`
    class to convert the customer instances to and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this viewset.
    Filtering is also available, with the ability to filter by is_active, equipment_type__name,
    fleet_code__code, and is_leased.
    """

    queryset = models.Trailer.objects.all()
    serializer_class = serializers.TrailerSerializer
    filterset_fields = (
        "status",
        "equipment_type__name",
        "fleet_code__code",
        "is_leased",
    )
    permission_classes = [CustomObjectPermissions]

    def get_queryset(self) -> QuerySet[models.Tractor]:
        user_org = self.request.user.organization_id  # type: ignore

        queryset = (
            self.queryset.filter(organization_id=user_org)
            .annotate(
                times_used=Count("movement", distinct=True),
                equip_type_name=F("equipment_type__name"),
            )
            .order_by("code")
            .all()
        )
        return queryset


class EquipmentManufacturerViewSet(viewsets.ModelViewSet):
    """A viewset for viewing and editing customer information in the system.

    The viewset provides default operations for creating, updating, and deleting customers,
    as well as listing and retrieving customers. It uses the `CustomerSerializer`
    class to convert the customer instances to and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this viewset.
    Filtering is also available, with the ability to filter by customer ID, name, and
    code.
    """

    queryset = models.EquipmentManufacturer.objects.all()
    serializer_class = serializers.EquipmentManufacturerSerializer
    permission_classes = [CustomObjectPermissions]

    def get_queryset(self) -> QuerySet[models.EquipmentManufacturer]:
        queryset = self.queryset.filter(
            organization__exact=self.request.user.organization  # type: ignore
        ).only(
            "id",
            "name",
            "organization_id",
            "description",
        )
        return queryset


class EquipmentMaintenancePlanViewSet(viewsets.ModelViewSet):
    """A viewset for viewing and editing customer information in the system.

    The viewset provides default operations for creating, updating, and deleting customers,
    as well as listing and retrieving customers. It uses the `CustomerSerializer`
    class to convert the customer instances to and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this viewset.
    Filtering is also available, with the ability to filter by customer ID, name, and
    code.
    """

    queryset = models.EquipmentMaintenancePlan.objects.all()
    serializer_class = serializers.EquipmentMaintenancePlanSerializer
    permission_classes = [CustomObjectPermissions]

    def get_queryset(self) -> QuerySet[models.EquipmentMaintenancePlan]:
        queryset = (
            self.queryset.filter(
                organization_id=self.request.user.organization_id  # type: ignore
            )
            .prefetch_related(
                Prefetch(
                    "equipment_types",
                    queryset=models.EquipmentType.objects.filter(
                        organization_id=self.request.user.organization_id  # type: ignore
                    ).only("id"),
                )
            )
            .only(
                "organization_id",
                "id",
                "miles",
                "by_engine_hours",
                "name",
                "by_distance",
                "months",
                "by_time",
                "engine_hours",
                "description",
            )
        )
        return queryset
