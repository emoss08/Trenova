# --------------------------------------------------------------------------------------------------
#  COPYRIGHT(c) 2024 Trenova                                                                       -
#                                                                                                  -
#  This file is part of Trenova.                                                                   -
#                                                                                                  -
#  The Trenova software is licensed under the Business Source License 1.1. You are granted the right
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

from dispatch import models
from utils.admin import GenericAdmin, GenericStackedInline


@admin.register(models.DispatchControl)
class DispatchControlAdmin(GenericAdmin[models.DispatchControl]):
    """
    Dispatch Control Admin
    """

    model: type[models.DispatchControl] = models.DispatchControl
    list_display = (
        "organization",
        "record_service_incident",
    )
    search_fields = ("organization",)


@admin.register(models.DelayCode)
class DelayCodeAdmin(GenericAdmin[models.DelayCode]):
    """
    Delay Code Admin
    """

    model: type[models.DelayCode] = models.DelayCode
    list_display = (
        "code",
        "description",
    )
    search_fields = ("code", "description")


@admin.register(models.FleetCode)
class FleetCodeAdmin(GenericAdmin[models.FleetCode]):
    """
    Fleet Code Admin
    """

    model: type[models.FleetCode] = models.FleetCode
    list_display = (
        "code",
        "description",
    )
    search_fields = ("code", "description")


@admin.register(models.CommentType)
class CommentTypeAdmin(GenericAdmin[models.CommentType]):
    """
    Comment Type admin
    """

    model: type[models.CommentType] = models.CommentType
    list_display = ("name",)
    search_fields = ("name",)


class RateBillingTableAdmin(GenericStackedInline[models.Rate, models.RateBillingTable]):
    """
    Rate Billing Table Admin
    """

    model: type[models.RateBillingTable] = models.RateBillingTable
    extra = 0
    exclude = ("organization",)
    autocomplete_fields = ("accessorial_charge",)


@admin.register(models.Rate)
class RateAdmin(GenericAdmin[models.Rate]):
    """
    Rate Admin
    """

    model: type[models.Rate] = models.Rate
    list_display = (
        "rate_number",
        "customer",
    )
    search_fields = ("rate_number",)
    inlines = (RateBillingTableAdmin,)


@admin.register(models.FeasibilityToolControl)
class FeasibilityToolControlAdmin(GenericAdmin[models.FeasibilityToolControl]):
    """
    Feasibility Tool Control Admin
    """

    autocomplete = False
    model: type[models.FeasibilityToolControl] = models.FeasibilityToolControl
    list_display = ("organization", "id")

    def has_delete_permission(
        self, request: HttpRequest, obj: models.FeasibilityToolControl | None = None
    ) -> bool:
        """Has permission to delete.

        Args:
            request (HttpRequest): Request object from the view function that called this method (if any).
            obj (models.FeasibilityToolControl | None): Object to be deleted (if any).

        Returns:
            bool: True if the user has permission to delete the given object, False otherwise.
        """
        return False
