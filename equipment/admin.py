# -*- coding: utf-8 -*-
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

from typing import Type

from django.contrib import admin

from .models import (
    Equipment,
    EquipmentMaintenancePlan,
    EquipmentManufacturer,
    EquipmentType,
    EquipmentTypeDetail,
)


@admin.register(EquipmentManufacturer)
class EquipmentManufacturerAdmin(admin.ModelAdmin):
    """
    Equipment Manufacturer Admin
    """

    model: Type[EquipmentManufacturer] = EquipmentManufacturer
    list_display: tuple[str, ...] = (
        "id",
        "description",
    )
    search_fields: tuple[str, ...] = (
        "id",
        "description",
    )
    autocomplete_fields: tuple[str, ...] = ("organization",)


class EquipmentTypeDetailAdmin(admin.StackedInline):
    """
    Equipment Type Detail Admin
    """

    model: Type[EquipmentTypeDetail] = EquipmentTypeDetail
    can_delete: bool = False
    verbose_name_plural: str = "Equipment Type Details"
    fk_name: str = "equipment_type"
    extra: int = 0
    autocomplete_fields: tuple[str, ...] = ("equipment_type", "organization")


@admin.register(EquipmentType)
class EquipmentTypeAdmin(admin.ModelAdmin):
    """
    Equipment Type Admin
    """

    model: Type[EquipmentType] = EquipmentType
    list_display: tuple[str, ...] = ("name", "description")
    search_fields: tuple[str, ...] = ("name", "description")
    inlines: tuple[Type[EquipmentTypeDetailAdmin], ...] = (EquipmentTypeDetailAdmin,)
    autocomplete_fields: tuple[str, ...] = ("organization",)


@admin.register(Equipment)
class EquipmentAdmin(admin.ModelAdmin):
    """
    Equipment Admin
    """

    model: Type[Equipment] = Equipment
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
    autocomplete_fields: tuple[str, ...] = (
        "equipment_type",
        "organization",
        "manufacturer",
    )
    fieldsets = (
        (
            None,
            {
                "fields": (
                    "is_active",
                    "organization",
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
class EquipmentMaintenancePlanAdmin(admin.ModelAdmin):
    """
    Equipment Maintenance Plan Admin
    """

    model: Type[EquipmentMaintenancePlan] = EquipmentMaintenancePlan
    list_display: tuple[str, ...] = (
        "id",
        "description",
    )
    search_fields: tuple[str, ...] = (
        "id",
        "description",
        "equipment_types",
    )
    autocomplete_fields: tuple[str, ...] = (
        "equipment_types",
        "organization",
    )
    fieldsets = (
        (None, {"fields": ("organization", "id", "equipment_types", "description")}),
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
