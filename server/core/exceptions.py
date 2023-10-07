# --------------------------------------------------------------------------------------------------
#  COPYRIGHT(c) 2023 MONTA                                                                         -
#                                                                                                  -
#  This file is part of Monta.                                                                     -
#                                                                                                  -
#  The Monta software is licensed under the Business Source License 1.1. You are granted the right -
#  to copy, modify, and redistribute the software, but only for non-production use or with a total -
#  of less than three server instances. Starting from the Change Date (November 16, 2026), the     -
#  software will be made available under version 2 or later of the GNU General Public License.     -
#  If you use the software in violation of this license, your rights under the license will be     -
#  terminated automatically. The software is provided "as is," and the Licensor disclaims all      -
#  warranties and conditions. If you use this license's text or the "Business Source License" name -
#  and trademark, you must comply with the Licensor's covenants, which include specifying the      -
#  Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use     -
#  Grant, and not modifying the license in any other way.                                          -
# --------------------------------------------------------------------------------------------------
import typing

from django.core.exceptions import ValidationError
from drf_standardized_errors.formatter import ExceptionFormatter
from drf_standardized_errors.handler import ExceptionHandler
from drf_standardized_errors.types import ErrorResponse
from rest_framework import exceptions


class CustomExceptionFormatter(ExceptionFormatter):
    @staticmethod
    def snake_to_camel(snake_str: str) -> str:
        if snake_str is None:
            return None  # type: ignore
        components = snake_str.split("_")
        return components[0] + "".join(x.title() for x in components[1:])

    def format_error_response(
        self, error_response: ErrorResponse
    ) -> dict[str, typing.Any]:
        error_response.type = self.snake_to_camel(error_response.type)
        for error in error_response.errors:
            error.attr = self.snake_to_camel(error.attr)
        return super().format_error_response(error_response)


class CustomExceptionHandler(ExceptionHandler):
    def convert_known_exceptions(self, exc: Exception) -> Exception:
        if isinstance(exc, ValidationError):
            return exceptions.ValidationError(detail=exc.message_dict)
        return super().convert_known_exceptions(exc)


class ServiceException(Exception):
    """
    Base Service Exception for all services.
    """


class CommandCallException(Exception):
    """
    Base Command call exception.

    This exception is raised when a django command is called with invalid arguments,
    or when the command fails to execute.
    """
