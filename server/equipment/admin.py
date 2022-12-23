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

from equipment import models
from utils.admin import GenericAdmin, GenericStackedInline


@admin.register(models.EquipmentManufacturer)
class EquipmentManufacturerAdmin(GenericAdmin[models.EquipmentManufacturer]):
    """
    Equipment Manufacturer Admin
    """

    model: type[models.EquipmentManufacturer] = models.EquipmentManufacturer
    list_display: tuple[str, ...] = (
        "id",
        "description",
    )
    search_fields: tuple[str, ...] = (
        "id",
        "description",
    )


class EquipmentTypeDetailAdmin(
    GenericStackedInline[models.EquipmentType, models.EquipmentTypeDetail]
):
    """
    Equipment Type Detail Admin
    """

    model: type[models.EquipmentTypeDetail] = models.EquipmentTypeDetail
    can_delete = False
    verbose_name_plural = "Equipment Type Details"
    fk_name = "equipment_type"


@admin.register(models.EquipmentType)
class EquipmentTypeAdmin(GenericAdmin[models.EquipmentType]):
    """
    Equipment Type Admin
    """

    model: type[models.EquipmentType] = models.EquipmentType
    list_display: tuple[str, ...] = ("id", "description")
    search_fields: tuple[str, ...] = ("id", "description")
    inlines = (EquipmentTypeDetailAdmin,)


@admin.register(models.Equipment)
class EquipmentAdmin(GenericAdmin[models.Equipment]):
    """
    Equipment Admin
    """

    model: type[models.Equipment] = models.Equipment
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


@admin.register(models.EquipmentMaintenancePlan)
class EquipmentMaintenancePlanAdmin(GenericAdmin[models.EquipmentMaintenancePlan]):
    """
    Equipment Maintenance Plan Admin
    """

    model: type[models.EquipmentMaintenancePlan] = models.EquipmentMaintenancePlan
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
