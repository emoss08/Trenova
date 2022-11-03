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

from .models import Location, LocationCategory, LocationComment, LocationContact


@admin.register(LocationCategory)
class LocationCategoryAdmin(admin.ModelAdmin[LocationCategory]):
    """
    Location Category Admin
    """

    list_display: tuple[str, ...] = (
        "name",
        "description",
    )
    search_fields: tuple[str, ...] = (
        "name",
        "description",
    )


class LocationCommentAdmin(admin.StackedInline):
    """
    Location Comment Admin
    """

    model: Type[LocationComment] = LocationComment
    verbose_name_plural = "Location Comments"
    fk_name = "location"
    extra = 0
    autocomplete_fields = ("location",)


class LocationContactAdmin(admin.StackedInline):
    """
    Location Contact Admin
    """

    model: Type[LocationContact] = LocationContact
    verbose_name_plural = "Location Contacts"
    fk_name = "location"
    extra = 0
    autocomplete_fields = ("location",)


@admin.register(Location)
class LocationAdmin(admin.ModelAdmin[Location]):
    """
    Location Admin
    """

    list_display: tuple[str, ...] = (
        "id",
        "category",
        "address_line_1",
        "city",
        "state",
        "zip_code",
    )
    list_filter: tuple[str, ...] = (
        "depot",
        "category",
    )
    search_fields: tuple[str, ...] = (
        "address_line_1",
        "city",
        "state",
        "zip_code",
    )
    autocomplete_fields = ("category", "depot", "organization")
    inlines = (
        LocationCommentAdmin,
        LocationContactAdmin,
    )
