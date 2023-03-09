# --------------------------------------------------------------------------------------------------
#  COPYRIGHT(c) 2023 MONTA                                                                         -
#                                                                                                  -
#  This file is part of Monta.                                                                     -
#                                                                                                  -
#  Monta is free software: you can redistribute it and/or modify                                   -
#  it under the terms of the GNU General Public License as published by                            -
#  the Free Software Foundation, either version 3 of the License, or                               -
#  (at your option) any later version.                                                             -
#                                                                                                  -
#  Monta is distributed in the hope that it will be useful,                                        -
#  but WITHOUT ANY WARRANTY; without even the implied warranty of                                  -
#  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the                                   -
#  GNU General Public License for more details.                                                    -
#                                                                                                  -
#  You should have received a copy of the GNU General Public License                               -
#  along with Monta.  If not, see <https://www.gnu.org/licenses/>.                                 -
# --------------------------------------------------------------------------------------------------
from collections.abc import Iterable

from utils.types import MODEL_UUID
from accounts import models

def get_users_by_organization_id(
    *, organization_id: MODEL_UUID
) -> Iterable[models.User]:
    """
    Get Users by organization_id
    Args:
        organization_id (MODEL_UUID): Organization ID
        fields (list[str]): Fields to select
        select_related (list[str]): Select related fields

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