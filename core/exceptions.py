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

from django.core.exceptions import ValidationError

from rest_framework.views import exception_handler
from rest_framework.response import Response


def django_error_handler(exc: Any, context: Any) -> Response:
    """ Django error handler

    Args:
        exc (Exception): Exception
        context ():

    Returns:
        Response: Response
    """

    response: Response = exception_handler(exc, context)
    if response is None and isinstance(exc, ValidationError):
        return Response(status=400, data=exc.message_dict)
    return response