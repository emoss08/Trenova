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

from dispatch import models
from utils.admin import GenericAdmin


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
