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

from accounting import models, serializers
from utils.views import OrganizationMixin


class GeneralLedgerAccountViewSet(OrganizationMixin):
    """
    General Ledger Account ViewSet
    """

    serializer_class = serializers.GeneralLedgerAccountSerializer
    queryset = models.GeneralLedgerAccount.objects.all()
    filterset_fields = [
        "is_active",
        "account_number",
        "account_type",
    ]


class RevenueCodeViewSet(OrganizationMixin):
    """
    Revenue Code ViewSet
    """

    serializer_class = serializers.RevenueCodeSerializer
    queryset = models.RevenueCode.objects.all()
    filterset_fields = ["code"]


class DivisionCodeViewSet(OrganizationMixin):
    """
    Division Code ViewSet
    """

    serializer_class = serializers.DivisionCodeSerializer
    queryset = models.DivisionCode.objects.all()
    filterset_fields = ["code", "is_active"]

    def get_queryset(self) -> QuerySet[models.DivisionCode]:
        """Filter the queryset to only include the current user's organization

        Returns:
            QuerySet[models.DivisionCode]: Filtered queryset
        """
        return (
            self.queryset.filter(organization=self.request.user.organization)  # type: ignore
            .select_related(
                "organization",
                "cash_account",
                "ap_account",
                "expense_account",
            )
            .only(
                "id",
                "is_active",
                "code",
                "description",
                "organization__id",
                "cash_account",
                "ap_account",
                "expense_account",
            )
        )
