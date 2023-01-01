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
from django.db.models import QuerySet
from django_filters.rest_framework import DjangoFilterBackend
from rest_framework import permissions

from commodities import models, serializers
from utils.views import OrganizationViewSet


class HazardousMaterialViewSet(OrganizationViewSet):
    """A viewset for viewing and editing hazardous materials in the system.

    The viewset provides default operations for creating, updating, and deleting hazardous materials,
    as well as listing and retrieving hazardous materials. It uses the `HazardousMaterialSerializer`
    class to convert the hazardous material instances to and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this viewset.
    Filtering is also available, with the ability to filter by hazardous material ID, name, and
    description.
    """

    queryset = models.HazardousMaterial.objects.all()
    serializer_class = serializers.HazardousMaterialSerializer
    permission_classes = (permissions.IsAuthenticated,)
    filter_backends = (DjangoFilterBackend,)
    filterset_fields = (
        "id",
        "name",
        "description",
    )


class CommodityViewSet(OrganizationViewSet):
    """A viewset for viewing and editing commodities in the system.

    The viewset provides default operations for creating, updating, and deleting commodities,
    as well as listing and retrieving commodities. It uses the `CommoditySerializer`
    class to convert the commodity instances to and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this viewset.
    Filtering is also available, with the ability to filter by commodity ID, name, and
    description.
    """

    queryset = models.Commodity.objects.all()
    serializer_class = serializers.CommoditySerializer
    permission_classes = (permissions.IsAuthenticated,)
    filter_backends = (DjangoFilterBackend,)
    filterset_fields = (
        "id",
        "name",
        "description",
    )

    def get_queryset(self) -> QuerySet[models.Commodity]:
        """
        Returns the queryset of commodities that are associated with the current user's organization.

        Returns:
            The queryset of commodities that are associated with the current user's organization.
        """

        return self.queryset.filter(
            organization=self.request.user.organization  # type: ignore
        ).select_related("hazmat", "organization")
