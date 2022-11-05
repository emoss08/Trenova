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

from .models import DelayCode, DispatchControl, FleetCode


@admin.register(DispatchControl)
class DispatchControlAdmin(admin.ModelAdmin[DispatchControl]):
    """
    Dispatch Control Admin
    """

    model: Type[DispatchControl] = DispatchControl
    list_display = (
        "organization",
        "record_service_incident",
    )
    search_fields = ("organization",)
    autocomplete_fields = ("organization",)


@admin.register(DelayCode)
class DelayCodeAdmin(admin.ModelAdmin[DelayCode]):
    """
    Delay Code Admin
    """

    model: Type[DelayCode] = DelayCode
    list_display = (
        "code",
        "description",
    )
    search_fields = ("code", "description")
    autocomplete_fields = ("organization",)


@admin.register(FleetCode)
class FleetCodeAdmin(admin.ModelAdmin[FleetCode]):
    """
    Fleet Code Admin
    """

    model: Type[FleetCode] = FleetCode
    list_display = (
        "code",
        "description",
    )
    search_fields = ("code", "description")
    autocomplete_fields = ("organization",)
