# -*- coding: utf-8 -*-
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

from typing import Type

from django.db.models import QuerySet
from django_filters.rest_framework import DjangoFilterBackend  # type: ignore
from rest_framework import permissions, viewsets  # type: ignore
from rest_framework.filters import OrderingFilter, SearchFilter  # type: ignore

from .models import Equipment
from .serializers import EquipmentSerializer


class EquipmentViewSet(viewsets.ModelViewSet):
    """
    API endpoint that allows equipment to be viewed or edited.
    """

    queryset = Equipment.objects.all()
    serializer_class: Type[EquipmentSerializer] = EquipmentSerializer
    filterset_fields: tuple[str, ...] = ("equipment_type__name", "manufacturer")
    search_fields: tuple[str, ...] = ("id", "equipment_type__name", "manufacturer__id")
    ordering_fields: tuple[str, ...] = ("id", "equipment_type__name", "manufacturer__id")

    def get_queryset(self) -> QuerySet[Equipment]:
        """Get the queryset for this view.

        Filters the queryset to only include equipment for the requesting user's organization.

        Returns:

        """
        return self.queryset.filter(organization=self.request.user.profile.organization)
