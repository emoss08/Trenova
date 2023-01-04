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

from utils.views import OrganizationViewSet
from worker import models, serializers


class WorkerViewSet(OrganizationViewSet):
    """A viewset for viewing and editing workers in the system.

    The viewset provides default operations for creating, updating, and deleting workers,
    as well as listing and retrieving workers. It uses the `WorkerSerializer` class to
    convert the worker instances to and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this viewset.
    Filtering is also available, with the ability to filter by worker ID, first name, code,
    and last name.
    """

    queryset = models.Worker.objects.all()
    serializer_class = serializers.WorkerSerializer
    filter_backends = [DjangoFilterBackend]
    filterset_fields = ["id", "first_name", "code", "last_name"]

    def get_queryset(self) -> QuerySet[models.Worker]:
        """Returns a queryset of workers for the current user's organization.

        The queryset includes related fields such as profiles, manager(user), depot, organization,
        entered_by(user). It also prefetches related comments and contacts.

        Returns:
            QuerySet[models.Worker]: A queryset of workers for the current user's organization.
        """

        return self.queryset.select_related(
            "profiles",
            "manager",
            "depot",
            "organization",
            "entered_by",
            "entered_by__organization",
        ).prefetch_related("contacts", "comments")
