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
from utils.admin import GenericAdmin


@admin.register(models.EquipmentManufacturer)
class EquipmentManufacturerAdmin(GenericAdmin[models.EquipmentManufacturer]):
    """
    Equipment Manufacturer Admin
    """

    model = models.EquipmentManufacturer
    list_display = (
        "name",
        "description",
    )
    search_fields = (
        "name",
        "description",
    )


@admin.register(models.EquipmentType)
class EquipmentTypeAdmin(GenericAdmin[models.EquipmentType]):
    """
    Equipment Type Admin
    """

    model = models.EquipmentType
    list_display = ("name", "description")
    search_fields = ("name", "description")


@admin.register(models.Tractor)
class TractorAdmin(GenericAdmin[models.Tractor]):
    """
    Equipment Admin
    """

    model = models.Tractor
    list_display = (
        "code",
        "license_plate_number",
    )
    search_fields = (
        "code",
        "license_plate_number",
    )


@admin.register(models.Trailer)
class TrailerAdmin(GenericAdmin[models.Trailer]):
    model = models.Trailer
    list_display = (
        "code",
        "license_plate_number",
    )
    search_fields = (
        "code",
        "license_plate_number",
    )


@admin.register(models.EquipmentMaintenancePlan)
class EquipmentMaintenancePlanAdmin(GenericAdmin[models.EquipmentMaintenancePlan]):
    """
    Equipment Maintenance Plan Admin
    """

    model = models.EquipmentMaintenancePlan
    list_display = ("name",)
    search_fields = (
        "name",
        "equipment_types",
    )
    fieldsets = (
        (None, {"fields": ("name", "equipment_types")}),
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
