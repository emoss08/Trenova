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

from django.contrib import admin

from core.mixins import MontaAdminMixin, MontaStackedInlineMixin
from .models import (
    Equipment,
    EquipmentMaintenancePlan,
    EquipmentManufacturer,
    EquipmentType,
    EquipmentTypeDetail,
)


@admin.register(EquipmentManufacturer)
class EquipmentManufacturerAdmin(MontaAdminMixin[EquipmentManufacturer]):
    """
    Equipment Manufacturer Admin
    """

    model: type[EquipmentManufacturer] = EquipmentManufacturer
    list_display: tuple[str, ...] = (
        "id",
        "description",
    )
    search_fields: tuple[str, ...] = (
        "id",
        "description",
    )


class EquipmentTypeDetailAdmin(
    MontaStackedInlineMixin[EquipmentType, EquipmentTypeDetail]
):
    """
    Equipment Type Detail Admin
    """

    model: type[EquipmentTypeDetail] = EquipmentTypeDetail
    can_delete: bool = False
    verbose_name_plural: str = "Equipment Type Details"
    fk_name: str = "equipment_type"


@admin.register(EquipmentType)
class EquipmentTypeAdmin(MontaAdminMixin[EquipmentType]):
    """
    Equipment Type Admin
    """

    model: type[EquipmentType] = EquipmentType
    list_display: tuple[str, ...] = ("name", "description")
    search_fields: tuple[str, ...] = ("name", "description")
    inlines: tuple[type[EquipmentTypeDetailAdmin], ...] = (EquipmentTypeDetailAdmin,)


@admin.register(Equipment)
class EquipmentAdmin(MontaAdminMixin[Equipment]):
    """
    Equipment Admin
    """

    model: type[Equipment] = Equipment
    list_display: tuple[str, ...] = (
        "id",
        "description",
        "license_plate_number",
    )
    search_fields: tuple[str, ...] = (
        "id",
        "description",
        "license_plate_number",
    )
    fieldsets = (
        (
            None,
            {
                "fields": (
                    "is_active",
                    "id",
                    "equipment_type",
                    "description",
                )
            },
        ),
        (
            "Equipment Details",
            {
                "classes": ("collapse",),
                "fields": (
                    "license_plate_number",
                    "vin_number",
                    "manufacturer",
                    "model",
                    "model_year",
                    "state",
                    "leased",
                    "leased_date",
                    "primary_worker",
                    "secondary_worker",
                ),
            },
        ),
        (
            "Advanced Options",
            {
                "classes": ("collapse",),
                "fields": (
                    "hos_exempt",
                    "aux_power_unit_type",
                    "fuel_draw_capacity",
                    "num_of_axles",
                    "transmission_manufacturer",
                    "transmission_type",
                    "has_berth",
                    "has_electronic_engine",
                    "highway_use_tax",
                    "owner_operated",
                    "ifta_qualified",
                ),
            },
        ),
    )


@admin.register(EquipmentMaintenancePlan)
class EquipmentMaintenancePlanAdmin(MontaAdminMixin[EquipmentMaintenancePlan]):
    """
    Equipment Maintenance Plan Admin
    """

    model: type[EquipmentMaintenancePlan] = EquipmentMaintenancePlan
    list_display: tuple[str, ...] = (
        "id",
        "description",
    )
    search_fields: tuple[str, ...] = (
        "id",
        "description",
        "equipment_types",
    )
    fieldsets = (
        (None, {"fields": ("id", "equipment_types", "description")}),
        (
            "Schedule Details",
            {
                "classes": ("collapse",),
                "fields": (
                    "by_distance",
                    "by_time",
                    "by_engine_hours",
                    "miles",
                    "months",
                    "engine_hours",
                ),
            },
        ),
    )
