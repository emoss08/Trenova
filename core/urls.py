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

from django.urls import include, path
from rest_framework.schemas import get_schema_view

urlpatterns = [
    path(
        "openapi",
        get_schema_view(
            title="Monta", description="API for all things …", version="1.0.0"
        ),
        name="openapi-schema",
    ),
    path("users/", include("accounts.urls")),
]
