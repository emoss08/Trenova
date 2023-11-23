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

from location import models
from utils.admin import GenericAdmin, GenericStackedInline


@admin.register(models.LocationCategory)
class LocationCategoryAdmin(GenericAdmin[models.LocationCategory]):
    """
    Location Category Admin
    """

    list_display = (
        "name",
        "description",
    )
    search_fields = (
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


class LocationContactAdmin(
    GenericStackedInline[models.Location, models.LocationContact]
):
    """
    Location Contact Admin
    """

    model: type[models.LocationContact] = models.LocationContact
    verbose_name_plural = "Location Contacts"
    fk_name = "location"


@admin.register(models.Location)
class LocationAdmin(GenericAdmin[models.Location]):
    """
    Location Admin
    """

    list_display = (
        "code",
        "name",
        "location_category",
        "address_line_1",
        "city",
        "state",
        "zip_code",
    )
    list_filter = (
        "depot",
        "location_category",
    )
    search_fields = (
        "address_line_1",
        "city",
        "state",
        "zip_code",
    )
    inlines = (
        LocationCommentAdmin,
        LocationContactAdmin,
    )
