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
from collections.abc import Callable

from django.core.exceptions import ValidationError
from django.http import JsonResponse
from rest_framework import status


class BusinessUnitMiddleware:
    """
    Middleware to check if the user's business unit is paid.
    """

    def __init__(self, get_response: Callable) -> None:
        self.get_response = get_response

    def __call__(self, request):
        try:
            # existing middleware logic
            if not request.user.business_unit.paid:
                raise ValidationError(
                    "Your business unit is not paid, please contact your Account Manager."
                )

        except ValidationError as e:
            errors = e.messages  # this is a list
            return JsonResponse({"detail": errors[0]}, status=403)

        return self.get_response(request)
