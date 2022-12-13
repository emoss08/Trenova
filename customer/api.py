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

from customer import models, serializers
from utils.views import OrganizationViewSet


class CustomerViewSet(OrganizationViewSet):
    """A viewset for viewing and editing customer information in the system.

    The viewset provides default operations for creating, updating, and deleting customers,
    as well as listing and retrieving customers. It uses the `CustomerSerializer`
    class to convert the customer instances to and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this viewset.
    Filtering is also available, with the ability to filter by customer ID, name, and
    code.
    """

    queryset = models.Customer.objects.all()
    serializer_class = serializers.CustomerSerializer
    permission_classes = (permissions.IsAuthenticated,)
    filter_backends = [DjangoFilterBackend]
    filterset_fields = ("id", "code", "name")


class CustomerFuelTableViewSet(OrganizationViewSet):
    """A viewset for viewing and editing customer fuel table information in the system.

    The viewset provides default operations for creating, updating, and deleting customer
    fuel tables, as well as listing and retrieving customer fuel tables. It uses the
    `CustomerFuelTableSerializer` class to convert the customer fuel table instances to
    and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this viewset.
    Filtering is also available, with the ability to filter by customer fuel table ID,
    customer ID, and customer name.
    """

    queryset = models.CustomerFuelTable.objects.all()
    serializer_class = serializers.CustomerFuelTableSerializer
    permission_classes = (permissions.IsAuthenticated,)
    filter_backends = [DjangoFilterBackend]
    filterset_fields = (
        "id",
        "name",
    )

    def get_queryset(self) -> QuerySet[models.CustomerFuelTable]:
        """Get the queryset for the viewset.

        The queryset is filtered by the organization of the user making the request.

        Returns:
            The filtered queryset.
        """
        return (
            self.queryset.filter(
                organization=self.request.user.organization  # type: ignore
            )
            .select_related("organization")
            .prefetch_related("customer_fuel_table_details")
        )
