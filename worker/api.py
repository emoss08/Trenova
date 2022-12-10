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

from utils.views import OrganizationViewSet
from worker import models, serializers


class WorkerViewSet(OrganizationViewSet):
    """
    Worker View Set
    """

    queryset = models.Worker.objects.all()
    serializer_class = serializers.WorkerSerializer
    permission_classes = [permissions.IsAuthenticated]
    filter_backends = [DjangoFilterBackend]
    filterset_fields = ["id", "first_name", "code", "last_name"]

    def get_queryset(self) -> QuerySet[models.Worker]:
        """
        Get queryset
        """
        return (
            self.queryset.filter(organization=self.request.user.organization)  # type: ignore
            .select_related(
                "profiles",
                "manager",
                "depot",
                "organization",
            )
            .prefetch_related(
                "contacts",
                "comments",
            )
        )
