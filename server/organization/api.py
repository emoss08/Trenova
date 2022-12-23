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
from rest_framework import permissions, viewsets

from organization import models, serializers


class OrganizationViewSet(viewsets.ModelViewSet):
    """
    A viewset for viewing and editing organization instances.

    The viewset provides default operations for creating, updating, and deleting organizations,
    as well as listing and retrieving organizations. It uses the `OrganizationSerializer`
    class to convert the organization instances to and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this viewset.
    Filtering is also available, with the ability to filter by organization ID, name, and
    description.
    """

    permission_classes = [permissions.IsAuthenticated]
    serializer_class = serializers.OrganizationSerializer
    queryset = models.Organization.objects.all()

    def get_queryset(self) -> QuerySet[models.Organization]:
        """Filter the queryset to only include the current user

        Returns:
            QuerySet[models.Organization]: Filtered queryset
        """

        return self.queryset.prefetch_related(
            "depots",
            "depots__details",
        )


class DepotViewSet(viewsets.ModelViewSet):
    """
    Depot ViewSet to manage requests to the depot endpoint
    """

    permission_classes = [permissions.IsAuthenticated]
    serializer_class = serializers.DepotSerializer
    queryset = models.Depot.objects.all()

    def get_queryset(self) -> QuerySet[models.Depot]:
        """Filter the queryset to only include the current user

        Returns:
            QuerySet[models.Depot]: Filtered queryset
        """

        return self.queryset.filter(organization=self.request.user.organization.id)  # type: ignore


class DepartmentViewSet(viewsets.ModelViewSet):
    """
    Department ViewSet to manage requests to the department endpoint
    """

    permission_classes = [permissions.IsAuthenticated]
    serializer_class = serializers.DepartmentSerializer
    queryset = models.Department.objects.all()

    def get_queryset(self) -> QuerySet[models.Department]:
        """Filter the queryset to only include the current user

        Returns:
            QuerySet[models.Depot]: Filtered queryset
        """

        return self.queryset.filter(organization=self.request.user.organization.id)  # type: ignore
