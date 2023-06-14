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
from rest_framework import viewsets

from equipment import models, serializers


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

    def get_queryset(self) -> QuerySet[models.EquipmentType]:
        queryset = (
            self.queryset.filter(
                organization_id=self.request.user.organization_id  # type: ignore
            )
            .select_related("equipment_type_detail")
            .only(
                "organization_id",
                "name",
                "description",
                "id",
                "equipment_type_detail__id",
                "equipment_type_detail__idling_fuel_usage",
                "equipment_type_detail__height",
                "equipment_type_detail__exempt_from_tolls",
                "equipment_type_detail__equipment_type__id",
                "equipment_type_detail__variable_cost",
                "equipment_type_detail__weight",
                "equipment_type_detail__equipment_class",
                "equipment_type_detail__fixed_cost",
            )
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
        "is_active",
        "manufacturer",
        "equipment_type__name",
        "has_berth",
        "highway_use_tax",
    )

    def get_queryset(self) -> QuerySet[models.Tractor]:
        queryset = self.queryset.filter(
            organization_id=self.request.user.organization_id  # type: ignore
        ).only(
            "id",
            "organization__id",
            "equipment_type__id",
            "transmission_manufacturer",
            "num_of_axles",
            "leased",
            "leased_date",
            "ifta_qualified",
            "odometer",
            "manufacturer",
            "primary_worker__id",
            "vin_number",
            "engine_hours",
            "highway_use_tax",
            "is_active",
            "fuel_draw_capacity",
            "secondary_worker__id",
            "model_year",
            "state",
            "license_plate_number",
            "code",
            "hos_exempt",
            "transmission_type",
            "model",
            "owner_operated",
            "has_electronic_engine",
            "manufactured_date",
            "fleet__code",
            "has_berth",
            "description",
            "aux_power_unit_type",
        )
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
    filterset_fields = ("is_active", "equipment_type__name", "fleet__code", "is_leased")

    def get_queryset(self) -> QuerySet[models.Tractor]:
        queryset = self.queryset.filter(
            organization_id=self.request.user.organization_id  # type: ignore
        ).only(
            "id",
            "organization_id",
            "make",
            "lease_expiration_date",
            "fleet_id",
            "axles",
            "owner",
            "license_plate_number",
            "length",
            "model",
            "tag_identifier",
            "leased_date",
            "planning_comment",
            "license_plate_expiration_date",
            "code",
            "vin_number",
            "is_leased",
            "is_active",
            "state",
            "equipment_type_id",
            "width",
            "year",
            "last_inspection",
            "height",
            "license_plate_state",
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
