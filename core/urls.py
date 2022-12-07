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
from accounts import api as accounts_api

urlpatterns = [
    path(
        "token/provision/",
        accounts_api.TokenProvisionView.as_view(),
        name="token-provision",
    ),
    path("token/verify/", accounts_api.TokenVerifyView.as_view(), name="token-verify"),
    path("users/", include("accounts.urls")),
]
