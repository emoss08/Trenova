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

from typing import Any

from django.contrib import admin
from django.forms import ModelForm
from django.http import HttpRequest

from accounting import models
from utils.admin import GenericAdmin


@admin.register(models.GeneralLedgerAccount)
class GeneralLedgerAccountAdmin(GenericAdmin[models.GeneralLedgerAccount]):
    """
    General Ledger Account Admin
    """

    model = models.GeneralLedgerAccount
    list_display: tuple[str, ...] = (
        "account_number",
        "cash_flow_type",
        "account_sub_type",
        "account_classification",
    )
    search_fields: tuple[str, ...] = ("account_number",)

    def get_form(
        self,
        request: HttpRequest,
        obj: models.GeneralLedgerAccount | None = None,
        change: bool = False,
        **kwargs: Any,
    ) -> type[ModelForm[models.GeneralLedgerAccount]]:
        """Get Form for Model

        Args:
            change (bool): If the model is being changed
            request (HttpRequest): Request Object
            obj (Optional[GeneralLedgerAccount]): General Ledger Account Object
            **kwargs (Any): Keyword Arguments

        Returns:
            Type[ModelForm[Any]]: Form Class's
        """
        form = super().get_form(request, obj, **kwargs)
        form.base_fields["account_number"].widget.attrs["placeholder"] = "0000-00"
        form.base_fields["account_number"].widget.attrs["value"] = "0000-00"
        return form


@admin.register(models.RevenueCode)
class RevenueCodeAdmin(GenericAdmin[models.RevenueCode]):
    """
    Revenue Code Admin
    """

    model = models.RevenueCode
    list_display: tuple[str, ...] = (
        "code",
        "description",
    )
    search_fields: tuple[str, ...] = (
        "code",
        "description",
    )


@admin.register(models.DivisionCode)
class DivisionCodeAdmin(GenericAdmin[models.DivisionCode]):
    """
    Division Code Admin
    """

    model = models.DivisionCode
    list_display: tuple[str, ...] = (
        "code",
        "description",
    )
    search_fields: tuple[str, ...] = (
        "code",
        "description",
    )


@admin.register(models.Tag)
class TagAdmin(GenericAdmin[models.Tag]):
    """
    Tag Admin
    """

    model = models.Tag
    list_display: tuple[str, ...] = (
        "name",
        "description",
    )
    search_fields: tuple[str, ...] = (
        "name",
        "description",
    )


@admin.register(models.FinancialTransaction)
class FinancialTransactionAdmin(GenericAdmin[models.FinancialTransaction]):
    """
    FinancialTransaction Admin
    """

    model = models.FinancialTransaction
    list_display: tuple[str, ...] = (
        "transaction_number",
        "transaction_type",
        "amount",
    )
    search_fields: tuple[str, ...] = ("shipment__pro_number",)


@admin.register(models.AccountingControl)
class AccountingControlAdmin(GenericAdmin[models.AccountingControl]):
    """
    Billing Control Admin
    """

    model = models.AccountingControl
    list_display = ("organization", "auto_create_journal_entries")
    search_fields = ("organization", "auto_create_journal_entries")

    def has_delete_permission(
        self, request: HttpRequest, obj: models.AccountingControl | None = None
    ) -> bool:
        """Has Deleted Permission

        Args:
            request (HttpRequest): Request Object
            obj (Optional[AccountingControl]): Accounting Control Object

        Returns:
            bool: True if the user has permission to delete the given object, False otherwise.
        """
        return False


@admin.register(models.ReconciliationQueue)
class ReconciliationQueueAdmin(GenericAdmin[models.ReconciliationQueue]):
    """
    ReconciliationQueue Admin
    """

    model = models.ReconciliationQueue
    list_display: tuple[str, ...] = (
        "shipment",
        "resolved",
        "reason",
        "resolved_by",
    )
    search_fields: tuple[str, ...] = (
        "date_added",
        "reason",
    )
