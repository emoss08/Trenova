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

from organization import models
from utils.admin import GenericAdmin, GenericStackedInline


@admin.register(models.Organization)
class OrganizationAdmin(admin.ModelAdmin[models.Organization]):
    """
    Organization Admin
    """

    list_display: tuple[str, ...] = (
        "name",
        "scac_code",
        "org_type",
        "timezone",
    )
    list_filter: tuple[str, ...] = ("org_type",)
    search_fields: tuple[str, ...] = (
        "name",
        "scac_code",
    )


class DepotDetailInline(GenericStackedInline[models.Depot, models.DepotDetail]):
    """
    Depot Detail Admin
    """

    model: type[models.DepotDetail] = models.DepotDetail
    can_delete = False
    verbose_name_plural = "Depot Details"
    fk_name = "depot"


@admin.register(models.Depot)
class DepotAdmin(GenericAdmin[models.Depot]):
    """
    Depot Admin
    """

    list_display = (
        "name",
        "description",
    )
    list_filter = ("name",)
    search_fields = ("name",)
    inlines: tuple[type[DepotDetailInline]] = (DepotDetailInline,)


@admin.register(models.Department)
class DepartmentAdmin(GenericAdmin[models.Department]):
    """
    Department Admin
    """

    list_display = (
        "name",
        "description",
    )
    list_filter = ("name",)
    search_fields = ("name",)


@admin.register(models.EmailProfile)
class EmailProfileAdmin(GenericAdmin[models.EmailProfile]):
    """
    Email Profile Admin
    """

    list_display = (
        "name",
        "email",
    )
    search_fields = (
        "name",
        "email",
    )


@admin.register(models.EmailControl)
class EmailControlAdmin(GenericAdmin[models.EmailControl]):
    """
    Email Control Admin
    """

    autocomplete = False
