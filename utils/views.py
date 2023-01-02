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

from typing import TypeVar

from django.db.models import Model, QuerySet
from rest_framework import viewsets

_M = TypeVar("_M", bound=Model)


class OrganizationViewSet(viewsets.ModelViewSet):
    """
    Organization ViewSet to manage requests to the organization endpoint
    """

    def get_queryset(self) -> QuerySet[_M]:
        """Filter the queryset to only include the current user's organization

        Returns:

        """

        return self.queryset.filter(  # type: ignore
            organization=self.request.user.organization  # type: ignore
        ).select_related(
            "organization",
        )
