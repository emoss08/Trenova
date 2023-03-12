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
        "name",
        "description",
    )
    search_fields: tuple[str, ...] = (
        "name",
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
    list_display: tuple[str, ...] = ("name", "description")
    search_fields: tuple[str, ...] = ("name", "description")
    inlines = (EquipmentTypeDetailAdmin,)


@admin.register(models.Tractor)
class EquipmentAdmin(GenericAdmin[models.Tractor]):
    """
    Equipment Admin
    """

    model: type[models.Tractor] = models.Tractor
    list_display: tuple[str, ...] = (
        "code",
        "description",
        "license_plate_number",
    )
    search_fields: tuple[str, ...] = (
        "code",
        "description",
        "license_plate_number",
    )
    fieldsets = (
        (
            None,
            {
                "fields": (
                    "is_active",
                    "code",
                    "equipment_type",
                    "description",
                )
            },
        ),
        (
            "Tractor Details",
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
        "name",
        "description",
    )
    search_fields: tuple[str, ...] = (
        "name",
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
