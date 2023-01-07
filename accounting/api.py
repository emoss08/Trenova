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

from django_filters.rest_framework import DjangoFilterBackend
from rest_framework import permissions

from accounting import models, serializers
from utils.views import OrganizationViewSet


class GeneralLedgerAccountViewSet(OrganizationViewSet):
    """
    General Ledger Account ViewSet
    """

    serializer_class = serializers.GeneralLedgerAccountSerializer
    queryset = models.GeneralLedgerAccount.objects.all()
    permission_classes = [permissions.IsAuthenticated]
    filter_backends = [DjangoFilterBackend]
    filterset_fields = [
        "id",
        "is_active",
        "account_number",
        "account_type",
    ]


class RevenueCodeViewSet(OrganizationViewSet):
    """
    Revenue Code ViewSet
    """

    serializer_class = serializers.RevenueCodeSerializer
    queryset = models.RevenueCode.objects.all()
    filter_backends = [DjangoFilterBackend]
    filterset_fields = ["id", "code", "description"]


class DivisionCodeViewSet(OrganizationViewSet):
    """
    Division Code ViewSet
    """

    serializer_class = serializers.DivisionCodeSerializer
    queryset = models.DivisionCode.objects.all()
    filterset_fields = ["id", "code", "is_active"]
