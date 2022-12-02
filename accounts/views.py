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

from typing import Any

from rest_framework.request import Request
from rest_framework.response import Response
from rest_framework_simplejwt.views import TokenVerifyView


class GenericTokenVerifyView(TokenVerifyView):
    """
    If the token is valid return it back in the response
    """

    def post(self, request: Request, *args: Any, **kwargs: Any) -> Response:
        """Handle POST request

        Override the post method to return the token back in the response

        Args:
            request (Request): Request object
            *args (Any): Arguments
            **kwargs (Any): Keyword arguments

        Returns:
            Response: Token if valid
        """

        response: Response = super().post(request, *args, **kwargs)
        response.data["token"] = request.data["token"]
        return response
