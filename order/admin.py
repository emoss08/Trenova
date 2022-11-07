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

from core.generics.admin import GenericAdmin
from .models import (
    Commodity,
    HazardousMaterial,
    OrderControl,
    OrderType,
    QualifierCode,
    ReasonCode,
)


@admin.register(OrderType)
class OrderTypeAdmin(GenericAdmin[OrderType]):
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
class HazardousMaterialAdmin(GenericAdmin[HazardousMaterial]):
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
class CommodityAdmin(GenericAdmin[Commodity]):
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
class QualifierCodeAdmin(GenericAdmin[QualifierCode]):
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
class ReasonCodeAdmin(GenericAdmin[ReasonCode]):
    """
    Reason Code Admin
    """

    list_display = (
        "code",
        "description",
    )
    search_fields = ("code", "description")
    autocomplete_fields = ("organization",)


@admin.register(OrderControl)
class OrderControlAdmin(GenericAdmin[OrderControl]):
    """
    Order Control Admin
    """

    list_display = (
        "organization",
        "auto_rate_orders",
    )
    autocomplete_fields = ("organization",)
