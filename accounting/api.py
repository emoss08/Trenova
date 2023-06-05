# --------------------------------------------------------------------------------------------------
#  COPYRIGHT(c) 2023 MONTA                                                                         -
#                                                                                                  -
#  This file is part of Monta.                                                                     -
#                                                                                                  -
#  The Monta software is licensed under the Business Source License 1.1. You are granted the right -
#  to copy, modify, and redistribute the software, but only for non-production use or with a total -
#  of less than three server instances. Starting from the Change Date (November 16, 2026), the     -
#  software will be made available under version 2 or later of the GNU General Public License.     -
#  If you use the software in violation of this license, your rights under the license will be     -
#  terminated automatically. The software is provided "as is," and the Licensor disclaims all      -
#  warranties and conditions. If you use this license's text or the "Business Source License" name -
#  and trademark, you must comply with the Licensor's covenants, which include specifying the      -
#  Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use     -
#  Grant, and not modifying the license in any other way.                                          -
# --------------------------------------------------------------------------------------------------


from django.db.models import QuerySet
from rest_framework import viewsets

from accounting import models, serializers


class GeneralLedgerAccountViewSet(viewsets.ModelViewSet):
    """
    General Ledger Account ViewSet
    """

    serializer_class = serializers.GeneralLedgerAccountSerializer
    queryset = models.GeneralLedgerAccount.objects.all()
    filterset_fields = (
        "is_active",
        "account_number",
        "account_type",
        "cash_flow_type",
        "account_sub_type",
        "account_classification",
    )

    def get_queryset(self) -> QuerySet[models.GeneralLedgerAccount]:
        queryset = self.queryset.filter(
            organization=self.request.user.organization  # type: ignore
        ).only(
            "organization_id",
            "id",
            "cash_flow_type",
            "account_number",
            "description",
            "account_type",
            "account_sub_type",
            "account_classification",
            "is_active",
        )
        return queryset


class RevenueCodeViewSet(viewsets.ModelViewSet):
    """
    Revenue Code ViewSet
    """

    serializer_class = serializers.RevenueCodeSerializer
    queryset = models.RevenueCode.objects.all()
    filterset_fields = (
        "expense_account",
        "revenue_account",
    )

    def get_queryset(self) -> QuerySet[models.RevenueCode]:
        """Filter the queryset to only include the current user's organization

        Returns:
            QuerySet[models.RevenueCode]: Filtered queryset
        """
        queryset = self.queryset.filter(
            organization=self.request.user.organization  # type: ignore
        ).only(
            "id",
            "code",
            "description",
            "organization_id",
            "expense_account_id",
            "revenue_account_id",
        )
        return queryset


class DivisionCodeViewSet(viewsets.ModelViewSet):
    """
    Division Code ViewSet
    """

    serializer_class = serializers.DivisionCodeSerializer
    queryset = models.DivisionCode.objects.all()
    filterset_fields = (
        "is_active",
        "cash_account",
        "ap_account",
        "expense_account",
    )

    def get_queryset(self) -> QuerySet[models.DivisionCode]:
        """Filter the queryset to only include the current user's organization

        Returns:
            QuerySet[models.DivisionCode]: Filtered queryset
        """

        queryset = self.queryset.filter(
            organization=self.request.user.organization  # type: ignore
        ).only(
            "id",
            "is_active",
            "code",
            "description",
            "organization_id",
            "cash_account_id",
            "ap_account_id",
            "expense_account_id",
        )

        return queryset
