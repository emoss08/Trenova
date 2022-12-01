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

from django.contrib.auth import authenticate, login as auth_login
from django.http import HttpRequest, HttpResponse, JsonResponse
from django.shortcuts import render
from django.utils.decorators import method_decorator
from django.views import generic
from django.views.decorators.debug import sensitive_post_parameters


@method_decorator(sensitive_post_parameters("password"))
class LoginView(generic.View):
    """
    Login View
    """

    def get(self, request: HttpRequest) -> HttpResponse:
        """Handle Get request

        Render the template for login page

        Args:
            request (HttpRequest): Request object

        Returns:
            HttpResponse: Rendered template
        """
        return render(request, "accounts/login.html")

    def post(self, request: HttpRequest) -> JsonResponse:
        """Handle Post request

        Args:
            request (HttpRequest): Request object

        Returns:
            JsonResponse: JsonResponse message
            returned back to user.
        """
        username: str = request.POST["username"]
        password: str = request.POST["password"]
        user = authenticate(request, username=username, password=password)

        if user and user.is_active:
            auth_login(request, user)
            return JsonResponse({"message": "User logged in successfully!"})
        else:
            return JsonResponse({"message": "Check your credentials!"})
