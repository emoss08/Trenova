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

from django.http import HttpRequest, HttpResponse

from accounts.models import User
from organization.models import Organization


class AuthenticatedHttpRequest(HttpRequest):
    """
    Authenticated Http Request
    """

    user: User
    organization: Organization


class OrganizationMiddleware:
    """
    Append organization to request
    """

    def __init__(self, get_response):
        self.get_response = get_response

    def __call__(self, request: AuthenticatedHttpRequest) -> HttpResponse:
        request.organization = request.user.organization
        response = self.get_response(request)
        return response
