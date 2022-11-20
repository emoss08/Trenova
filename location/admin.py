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

from location import models
from utils.admin import GenericAdmin, GenericStackedInline


@admin.register(models.LocationCategory)
class LocationCategoryAdmin(GenericAdmin[models.LocationCategory]):
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


class LocationCommentAdmin(
    GenericStackedInline[models.Location, models.LocationComment]
):
    """
    Location Comment Admin
    """

    model: type[models.LocationComment] = models.LocationComment
    verbose_name_plural = "Location Comments"
    fk_name = "location"
    extra = 0


class LocationContactAdmin(
    GenericStackedInline[models.Location, models.LocationContact]
):
    """
    Location Contact Admin
    """

    model: type[models.LocationContact] = models.LocationContact
    verbose_name_plural = "Location Contacts"
    fk_name = "location"
    extra = 0


@admin.register(models.Location)
class LocationAdmin(GenericAdmin[models.Location]):
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
    inlines = (
        LocationCommentAdmin,
        LocationContactAdmin,
    )
