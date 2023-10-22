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

from django.db.models import Prefetch, QuerySet
from rest_framework import permissions, viewsets

from accounting import models, serializers
from core.permissions import CustomObjectPermissions


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
    permission_classes = [CustomObjectPermissions]

    def get_queryset(self) -> QuerySet[models.GeneralLedgerAccount]:
        """The get_queryset function is used to filter the queryset by organization_id.
        This is done so that a user can only see their own GeneralLedgerAccounts and not those of other organizations.
        The .only() function limits the fields returned in the response to just those specified.

        Args:
            self: Refer to the object itself

        Returns:
            QuerySet[models.GeneralLedgerAccount]: A queryset of generalledgeraccount objects
        """
        queryset = (
            self.queryset.filter(
                organization_id=self.request.user.organization_id  # type: ignore
            )
            .prefetch_related(
                Prefetch(
                    "tags",
                    queryset=models.Tag.objects.only(
                        "id",
                    ).filter(
                        organization_id=self.request.user.organization_id  # type: ignore
                    ),
                ),
            )
            .only(
                "organization_id",
                "business_unit_id",
                "id",
                "status",
                "account_number",
                "account_type",
                "cash_flow_type",
                "account_sub_type",
                "account_classification",
                "balance",
                "opening_balance",
                "closing_balance",
                "parent_account",
                "is_reconciled",
                "date_opened",
                "date_closed",
                "notes",
                "owner",
                "is_tax_relevant",
                "attachment",
                "interest_rate",
                "tags",
                "modified",
                "created",
            )
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
    permission_classes = [CustomObjectPermissions]

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
    permission_classes = [CustomObjectPermissions]
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
            QuerySet[models.DivisionCode]: A queryset of Division Code objects
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


class TagViewSet(viewsets.ModelViewSet):
    """
    Tag ViewSet
    """

    serializer_class = serializers.TagSerializer
    queryset = models.Tag.objects.all()
    search_fields = ("name",)
    permission_classes = [CustomObjectPermissions]
    filterset_fields = ("name",)

    def get_queryset(self) -> QuerySet[models.Tag]:
        """
        The get_queryset function is used to filter the queryset based on the user's organization.
        This is done by adding a filter to the queryset that only returns DivisionCodes with an
        organization_id equal to that of the current user. This ensures that users can only see
        DivisionCodes belonging to their own organization.

        Args:
            self: Refer to the current instance of a class

        Returns:
            QuerySet[models.DivisionCode]: A queryset of Division Code objects
        """
        queryset = self.queryset.filter(
            organization_id=self.request.user.organization_id  # type: ignore
        ).only(
            "id",
            "name",
            "description",
        )

        return queryset


class FinancialTransactionViewSet(viewsets.ModelViewSet):
    """
    FinancialTransaction ViewSet
    """

    serializer_class = serializers.FinancialTransactionSerializer
    queryset = models.FinancialTransaction.objects.all()
    search_fields = ("transaction_number",)
    permission_classes = [CustomObjectPermissions]
    filterset_fields = ("transaction_number",)

    def get_queryset(self) -> QuerySet[models.FinancialTransaction]:
        """
        The get_queryset function is used to filter the queryset based on the user's organization.
        This is done by adding a filter to the queryset that only returns DivisionCodes with an
        organization_id equal to that of the current user. This ensures that users can only see
        DivisionCodes belonging to their own organization.

        Args:
            self: Refer to the current instance of a class

        Returns:
            QuerySet[models.FinancialTransaction]: A queryset of Division Code objects
        """
        queryset = self.queryset.filter(
            organization_id=self.request.user.organization_id  # type: ignore
        ).only(
            "id",
            "transaction_number",
            "amount",
            "transaction_type",
            "date_created",
            "ledger_account_id",
            "shipment_id",
            "status",
            "created_by_id",
            "description",
            "external_reference",
        )

        return queryset


class ReconciliationQueueViewSet(viewsets.ModelViewSet):
    """
    ReconciliationQueue ViewSet
    """

    serializer_class = serializers.ReconciliationQueueSerializer
    queryset = models.ReconciliationQueue.objects.all()
    search_fields = ("transaction_number",)
    permission_classes = [CustomObjectPermissions]
    filterset_fields = ("transaction_number",)

    def get_queryset(self) -> QuerySet[models.ReconciliationQueue]:
        """
        The get_queryset function is used to filter the queryset based on the user's organization.
        This is done by adding a filter to the queryset that only returns DivisionCodes with an
        organization_id equal to that of the current user. This ensures that users can only see
        DivisionCodes belonging to their own organization.

        Args:
            self: Refer to the current instance of a class

        Returns:
            QuerySet[models.ReconciliationQueue]: A queryset of Division Code objects
        """
        queryset = self.queryset.filter(
            organization_id=self.request.user.organization_id  # type: ignore
        ).only(
            "id",
            "shipment_id",
            "reason",
            "date_added",
            "financial_transaction_id",
            "resolved",
            "resolved_by_id",
            "date_resolved",
            "notes",
        )

        return queryset


class AccountingControlViewSet(viewsets.ModelViewSet):
    """A viewset for viewing and editing Accounting Control in the system.

    The viewset provides default operations for updating, as well as listing and retrieving
    Accounting Control. It uses the `AccountingControlSerializer` class to convert the Accounting
    Control instances to and from JSON-formatted data.

    Only get, put, patch, head and options HTTP methods are allowed when using this viewset.
    Only Admin users are allowed to access the views provided by this viewset.
    """

    queryset = models.AccountingControl.objects.all()
    serializer_class = serializers.AccountingControlSerializer
    permission_classes = [permissions.IsAdminUser]
    http_method_names = ["get", "put", "patch", "head", "options"]

    def get_queryset(self) -> QuerySet[models.AccountingControl]:
        queryset = self.queryset.filter(
            organization_id=self.request.user.organization_id  # type: ignore
        ).only(
            "id",
            "organization_id",
            "auto_create_journal_entries",
            "journal_entry_criteria",
            "restrict_manual_journal_entries",
            "require_journal_entry_approval",
            "default_revenue_account",
            "default_expense_account",
            "enable_reconciliation_notifications",
            "reconciliation_notification_recipients",
            "reconciliation_threshold",
            "reconciliation_threshold_action",
            "halt_on_pending_reconciliation",
            "critical_processes",
        )
        return queryset
