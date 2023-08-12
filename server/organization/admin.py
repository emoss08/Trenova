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
from django.http import HttpRequest
from organization import models
from utils.admin import GenericAdmin, GenericStackedInline


@admin.register(models.BusinessUnit)
class BusinessUnitAdmin(admin.ModelAdmin[models.BusinessUnit]):
    """
    Business Unit Admin
    """

    list_display = ("name", "description")
    list_filter = ("name",)
    search_fields = ("name",)


@admin.register(models.Organization)
class OrganizationAdmin(admin.ModelAdmin[models.Organization]):
    """
    Organization Admin
    """

    list_display = ("name", "scac_code", "org_type", "timezone")
    list_filter = ("org_type",)
    search_fields = ("name", "scac_code")

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

    list_display = ("name", "description")
    list_filter = ("name",)
    search_fields = ("name",)
    inlines = (DepotDetailInline,)


@admin.register(models.Department)
class DepartmentAdmin(GenericAdmin[models.Department]):
    """
    Department Admin
    """

    list_display = ("name", "description")
    list_filter = ("name",)
    search_fields = ("name",)


@admin.register(models.EmailProfile)
class EmailProfileAdmin(GenericAdmin[models.EmailProfile]):
    """
    Email Profile Admin
    """

    list_display = ("name", "email")
    search_fields = ("name", "email")


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

    list_display = ("name", "rate")
    list_filter = ("name",)
    search_fields = ("name",)


@admin.register(models.TableChangeAlert)
class TableChangeAlertAdmin(GenericAdmin[models.TableChangeAlert]):
    """
    Table Change Alert Admin
    """

    list_display = ("name", "table", "source", "topic")

    list_filter = ("name", "source", "topic", "table")
    search_fields = ("name", "source", "topic", "table")


class NotificationSettingStackedInline(
    GenericStackedInline[models.NotificationType, models.NotificationSetting]
):
    """
    Notification Type Admin
    """

    model = models.NotificationSetting
    can_delete = False
    verbose_name_plural = "Notification Settings"


@admin.register(models.NotificationType)
class NotificationTypeAdmin(GenericAdmin[models.NotificationType]):
    """
    Notification Setting Admin
    """

    list_display = ("name", "description")

    list_filter = ("name",)
    search_fields = ("name",)
    inlines = (NotificationSettingStackedInline,)
