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
from collections.abc import Iterable

from django.http import HttpRequest

from utils.types import MODEL_UUID
from accounts import models


def get_users_by_organization_id(
    *, organization_id: MODEL_UUID
) -> Iterable[models.User]:
    """
    Get Users by organization_id
    Args:
        organization_id (MODEL_UUID): Organization ID

    Returns:
        Iterable[models.User]: Users

    Examples:
        >>> from accounts import selectors
        ... from organization.models import Organization
        ... organization = Organization.objects.first()
        ... users = selectors.get_users_by_organization_id(organization_id=organization.id)
        ... print(users)
        <QuerySet [<User: User object (1)>, <User: User object (2)>]>
    """

    return models.User.objects.filter(organization_id=organization_id)


def get_user_auth_token_from_request(*, request: HttpRequest) -> str:
    """
    Retrieve or create an authentication token for a user.

    Args:
        request (HttpRequest): The HTTP request object containing the user for whom to retrieve the token.

    Returns:
        Token: An authentication token object associated with the specified user.

    Raises:
        Token.DoesNotExist: If no token exists for the specified user.
    """

    token, _ = models.Token.objects.get_or_create(user=request.user)
    return token.key
