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
from django.http import HttpRequest

from organization import models
from utils.admin import GenericAdmin, GenericStackedInline


@admin.register(models.Organization)
class OrganizationAdmin(admin.ModelAdmin[models.Organization]):
    """
    Organization Admin
    """

    list_display = (
        "name",
        "scac_code",
        "org_type",
        "timezone",
    )
    list_filter = ("org_type",)
    search_fields = (
        "name",
        "scac_code",
    )

    def has_delete_permission(
            self, request: HttpRequest, obj: models.EmailLog | None = None
    ) -> bool:
        """Has permission to delete.

        Args:
            request (HttpRequest): Request object from the view function that called this method (if any).
            obj (models.EmailLog | None): Object to be deleted (if any).

        Returns:
            bool: True if the user has permission to delete the given object, False otherwise.
        """
        return False


class DepotDetailInline(GenericStackedInline[models.Depot, models.DepotDetail]):
    """
    Depot Detail Admin
    """

    model = models.DepotDetail
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
    inlines = (DepotDetailInline,)


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


@admin.register(models.EmailLog)
class EmailLogAdmin(admin.ModelAdmin[models.EmailLog]):
    """
    Email Log Admin
    """

    list_display = ("to_email", "subject", "created")
    readonly_fields = ("to_email", "subject", "error", "created")
    search_fields = ("subject", "to_email", "created")

    def has_delete_permission(
        self, request: HttpRequest, obj: models.EmailLog | None = None
    ) -> bool:
        """Has permission to delete.

        Args:
            request (HttpRequest): Request object from the view function that called this method (if any).
            obj (models.EmailLog | None): Object to be deleted (if any).

        Returns:
            bool: True if the user has permission to delete the given object, False otherwise.
        """
        return False

    def has_add_permission(self, request: HttpRequest) -> bool:
        """Has permissions to add.

        Args:
            request (HttpRequest): Request object from the view function that called this method (if any).

        Returns:
            bool: True if the user has permission to add an object, False otherwise.
        """
        return False


@admin.register(models.TaxRate)
class TaxRateAdmin(GenericAdmin[models.TaxRate]):
    """
    Tax Rate Admin
    """

    list_display = (
        "name",
        "rate",
    )
    list_filter = ("name",)
    search_fields = ("name",)
