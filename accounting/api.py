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
    search_fields = (
        "account_number",
        "status",
        "account_type",
        "cash_flow_type",
        "account_sub_type",
        "account_classification",
    )
    filterset_fields = (
        "status",
        "account_number",
        "account_type",
        "cash_flow_type",
        "account_sub_type",
        "account_classification",
    )

    def get_queryset(self) -> QuerySet[models.GeneralLedgerAccount]:
        """The get_queryset function is used to filter the queryset by organization_id.
        This is done so that a user can only see their own GeneralLedgerAccounts and not those of other organizations.
        The .only() function limits the fields returned in the response to just those specified.

        Args:
            self: Refer to the object itself

        Returns:
            QuerySet[models.GeneralLedgerAccount]: A queryset of generalledgeraccount objects
        """
        queryset = self.queryset.filter(
            organization_id=self.request.user.organization_id  # type: ignore
        ).only(
            "organization_id",
            "id",
            "cash_flow_type",
            "account_number",
            "description",
            "account_type",
            "account_sub_type",
            "account_classification",
            "status",
        )
        return queryset


class RevenueCodeViewSet(viewsets.ModelViewSet):
    """
    Revenue Code ViewSet
    """

    serializer_class = serializers.RevenueCodeSerializer
    queryset = models.RevenueCode.objects.all()
    search_fields = ("code",)
    filterset_fields = (
        "expense_account",
        "revenue_account",
    )

    def get_queryset(self) -> QuerySet[models.RevenueCode]:
        """The get_queryset function is used to filter the queryset by organization_id.
        This is done so that a user can only see revenue codes for their own organization.
        The only function returns a list of fields that are needed in the serializer,
        and not all fields from the model.

        Args:
            self: Access the attributes and methods of the class in python

        Returns:
            A queryset of revenuecode objects
        """
        queryset = self.queryset.filter(
            organization_id=self.request.user.organization_id  # type: ignore
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
    search_fields = ("code", "status")

    filterset_fields = (
        "status",
        "cash_account",
        "ap_account",
        "expense_account",
    )

    def get_queryset(self) -> QuerySet[models.DivisionCode]:
        """
        The get_queryset function is used to filter the queryset based on the user's organization.
        This is done by adding a filter to the queryset that only returns DivisionCodes with an
        organization_id equal to that of the current user. This ensures that users can only see
        DivisionCodes belonging to their own organization.

        Args:
            self: Refer to the current instance of a class

        Returns:
            QuerySet[models.DivisionCode]: A queryset of divisioncode objects
        """
        queryset = self.queryset.filter(
            organization_id=self.request.user.organization_id  # type: ignore
        ).only(
            "id",
            "status",
            "code",
            "description",
            "organization_id",
            "cash_account_id",
            "ap_account_id",
            "expense_account_id",
        )

        return queryset
