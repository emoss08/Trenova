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

from commodities import models
from utils.admin import GenericAdmin


@admin.register(models.HazardousMaterial)
class HazardousMaterialAdmin(GenericAdmin[models.HazardousMaterial]):
    """
    Hazardous Material Admin
    """

    list_display = (
        "name",
        "description",
    )
    search_fields = ("name", "description")


@admin.register(models.Commodity)
class CommodityAdmin(GenericAdmin[models.Commodity]):
    """
    Commodity Admin
    """

    list_display = (
        "name",
        "description",
    )
    search_fields = ("name", "description")
    fieldsets = (
        (None, {"fields": ("name", "description")}),
        (
            "Hazmat Information",
            {
                "classes": ("collapse",),
                "fields": (
                    "min_temp",
                    "max_temp",
                    "set_point_temp",
                    "unit_of_measure",
                    "hazmat",
                    "is_hazmat",
                ),
            },
        ),
    )
