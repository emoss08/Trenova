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

from django.contrib import admin

from .models import Commodity, HazardousMaterial, OrderType, QualifierCode, ReasonCode


@admin.register(OrderType)
class OrderTypeAdmin(admin.ModelAdmin[OrderType]):
    """
    Order Type Admin
    """

    list_display = (
        "name",
        "description",
    )
    search_fields = ("name", "description")
    autocomplete_fields = ("organization",)


@admin.register(HazardousMaterial)
class HazardousMaterialAdmin(admin.ModelAdmin[HazardousMaterial]):
    """
    Hazardous Material Admin
    """

    list_display = (
        "name",
        "description",
    )
    search_fields = ("name", "description")
    autocomplete_fields = ("organization",)


@admin.register(Commodity)
class CommodityAdmin(admin.ModelAdmin[Commodity]):
    """
    Commodity Admin
    """

    list_display = (
        "name",
        "description",
    )
    search_fields = ("name", "description")
    autocomplete_fields = ("organization", "hazmat")
    fieldsets = (
        (None, {"fields": ("organization", "name", "description")}),
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


@admin.register(QualifierCode)
class QualifierCodeAdmin(admin.ModelAdmin[QualifierCode]):
    """
    Qualifier Code Admin
    """

    list_display = (
        "code",
        "description",
    )
    search_fields = ("code", "description")
    autocomplete_fields = ("organization",)


@admin.register(ReasonCode)
class ReasonCodeAdmin(admin.ModelAdmin[ReasonCode]):
    """
    Reason Code Admin
    """

    list_display = (
        "code",
        "description",
    )
    search_fields = ("code", "description")
    autocomplete_fields = ("organization",)
