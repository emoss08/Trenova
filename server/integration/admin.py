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

from integration import models
from utils.admin import GenericAdmin


@admin.register(models.Integration)
class IntegrationAdmin(GenericAdmin[models.Integration]):
    """
    Integration Admin
    """

    fieldsets = (
        (None, {"fields": ("name", "auth_type", "is_active")}),
        ("Basic Credentials", {"fields": ("auth_token", "client_id", "client_secret")}),
        (
            "Advanced Credentials",
            {"classes": ("collapse",), "fields": ("login_url", "username", "password")},
        ),
    )
    search_fields = ("name",)
