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

from core.mixins import MontaAdminMixin
from .models import DelayCode, DispatchControl, FleetCode


@admin.register(DispatchControl)
class DispatchControlAdmin(MontaAdminMixin[DispatchControl]):
    """
    Dispatch Control Admin
    """

    model: type[DispatchControl] = DispatchControl
    list_display = (
        "organization",
        "record_service_incident",
    )
    search_fields = ("organization",)


@admin.register(DelayCode)
class DelayCodeAdmin(MontaAdminMixin[DelayCode]):
    """
    Delay Code Admin
    """

    model: type[DelayCode] = DelayCode
    list_display = (
        "code",
        "description",
    )
    search_fields = ("code", "description")


@admin.register(FleetCode)
class FleetCodeAdmin(MontaAdminMixin[FleetCode]):
    """
    Fleet Code Admin
    """

    model: type[FleetCode] = FleetCode
    list_display = (
        "code",
        "description",
    )
    search_fields = ("code", "description")
