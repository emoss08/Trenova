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

from django.contrib.auth import mixins
from django.http import HttpRequest, HttpResponse
from django.shortcuts import render
from django.views import generic


class HomeView(mixins.LoginRequiredMixin, generic.View):
    """
    Home View
    """

    def get(self, request: HttpRequest) -> HttpResponse:
        """Handle Get request

        Render the template for home page

        Args:
            request (HttpRequest): Request object

        Returns:
            HttpResponse: Rendered template
        """
        return render(request, "core/home.html")
