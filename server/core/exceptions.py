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
import logging
import typing
from functools import lru_cache

from django.core.exceptions import ValidationError
from drf_standardized_errors.formatter import ExceptionFormatter
from drf_standardized_errors.handler import ExceptionHandler
from drf_standardized_errors.types import ErrorResponse
from rest_framework import exceptions


class CustomExceptionFormatter(ExceptionFormatter):
    # Logger for debugging and monitoring
    logger = logging.getLogger(__name__)

    @staticmethod
    @lru_cache(maxsize=128)  # Cache to optimize repeated transformations
    def snake_to_camel(snake_str: str) -> str:
        # Handling unexpected inputs gracefully
        if not snake_str:
            CustomExceptionFormatter.logger.warning(
                f"Received invalid input: {snake_str}"
            )
            return snake_str

        parts = snake_str.split(".")
        transformed_parts = []

        for part in parts:
            # Check for snake_case and convert to camelCase
            if "_" in part:
                transformed_parts.append(
                    CustomExceptionFormatter._convert_snake_to_camel(part)
                )
            else:
                # Keep as is or convert last part to lowercase after numeric index
                transformed_parts.append(
                    CustomExceptionFormatter._handle_numeric_index(
                        part, transformed_parts
                    )
                )

        return ".".join(transformed_parts)

    @staticmethod
    def _convert_snake_to_camel(part: str) -> str:
        """Converts snake_case to camelCase."""
        sub_parts = part.split("_")
        return sub_parts[0] + "".join(x.title() for x in sub_parts[1:])

    @staticmethod
    def _handle_numeric_index(part: str, transformed_parts: list) -> str:
        """Handles numeric index in the attribute name."""
        if transformed_parts and transformed_parts[-1].isdigit():
            return part.lower()
        return part

    def format_error_response(
        self, error_response: ErrorResponse
    ) -> dict[str, typing.Any]:
        error_response.type = self.snake_to_camel(error_response.type)  # type: ignore
        for error in error_response.errors:
            error.attr = self.snake_to_camel(error.attr)  # type: ignore
        return super().format_error_response(error_response)


class CustomExceptionHandler(ExceptionHandler):
    def convert_known_exceptions(self, exc: Exception) -> Exception:
        if isinstance(exc, ValidationError):
            # Ensure that the exception's message_dict is properly formatted
            formatted_messages = {
                self.snake_to_camel(k): v for k, v in exc.message_dict.items()
            }
            return exceptions.ValidationError(detail=formatted_messages)
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
