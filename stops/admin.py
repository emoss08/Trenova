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

from stops import models
from utils.admin import GenericAdmin


@admin.register(models.QualifierCode)
class QualifierCodeAdmin(GenericAdmin[models.QualifierCode]):
    """
    Qualifier Code Admin
    """

    list_display = (
        "code",
        "description",
    )
    search_fields = ("code", "description")


@admin.register(models.Stop)
class StopAdmin(GenericAdmin[models.Stop]):
    """
    Stop Admin
    """

    list_display = (
        "status",
        "movement",
        "stop_type",
        "sequence",
        "location",
        "address_line",
    )
    search_fields = ("id",)


@admin.register(models.ServiceIncident)
class ServiceIncidentAdmin(GenericAdmin[models.ServiceIncident]):
    """
    Service Incident Admin
    """

    list_display = (
        "movement",
        "stop",
        "delay_code",
        "delay_reason",
        "delay_time",
    )
    search_fields = ("id",)
