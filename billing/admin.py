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

from django.contrib import admin
from django.http import HttpRequest

from billing.models import (
    AccessorialCharge,
    BillingControl,
    ChargeType,
    DocumentClassification,
    BillingHistory,
    BillingQueue,
    BillingException,
    BillingTransferLog,
)
from utils.admin import GenericAdmin


@admin.register(BillingQueue)
class BillingQueueAdmin(GenericAdmin[BillingQueue]):
    """
    Billing Queue Admin
    """

    model: type[BillingQueue] = BillingQueue
    list_display = (
        "order",
        "bol_number",
        "invoice_number",
    )
    search_fields = ("invoice_number", "order", "bol_number")


@admin.register(BillingHistory)
class BillingHistoryAdmin(GenericAdmin[BillingHistory]):
    """
    Billing History Admin
    """

    model: type[BillingHistory] = BillingHistory
    list_display = (
        "order",
        "bol_number",
        "invoice_number",
    )
    search_fields = ("invoice_number", "order", "bol_number")

    def has_delete_permission(
        self, request: HttpRequest, obj: BillingHistory | None = None
    ) -> bool:
        """Has permission to delete.

        Args:
            request (HttpRequest): Request object from the view function that called this method (if any).
            obj (BillingHistory | None): Object to be deleted (if any).

        Returns:
            bool: True if the user has permission to delete the given object, False otherwise.
        """

        return bool(request.user.organization.billing_control.remove_billing_history)  # type: ignore

    def has_add_permission(self, request: HttpRequest) -> bool:
        """Has permissions to add.

        Args:
            request (HttpRequest): Request object from the view function that called this method (if any).

        Returns:
            bool: True if the user has permission to add an object, False otherwise.
        """
        return False


@admin.register(BillingTransferLog)
class BillingTransferLogAdmin(GenericAdmin[BillingTransferLog]):
    """
    Billing Transfer Log Admin
    """

    model: type[BillingTransferLog] = BillingTransferLog
    list_display = (
        "order",
        "transferred_at",
        "transferred_by",
    )
    search_fields = ("invoice_number", "transferred_by", "transferred_at")
    readonly_fields = ("transferred_at", "transferred_by", "order")

    def has_delete_permission(
        self, request: HttpRequest, obj: BillingTransferLog | None = None
    ) -> bool:
        """Has permission to delete.

        Args:
            request (HttpRequest): Request object from the view function that called this method (if any).
            obj (BillingTransferLog | None): Object to be deleted (if any).

        Returns:
            bool: True if the user has permission to delete the given object, False otherwise.
        """
        return False

    def has_add_permission(self, request: HttpRequest) -> bool:
        """Has permissions to add.

        Args:
            request (HttpRequest): Request object from the view function that called this method (if any).

        Returns:
            bool: True if the user has permission to add an object, False otherwise.
        """
        return False


@admin.register(BillingException)
class BillingExceptionAdmin(GenericAdmin[BillingException]):
    """
    Billing Exception Admin
    """

    model = BillingException
    list_display = (
        "order",
        "exception_type",
    )
    search_fields = (
        "exception_type",
        "order",
    )


@admin.register(BillingControl)
class BillingControlAdmin(GenericAdmin[BillingControl]):
    """
    Billing Control Admin
    """

    model: type[BillingControl] = BillingControl
    list_display = ("organization", "auto_bill_orders")
    search_fields = ("organization", "auto_bill_orders")


@admin.register(DocumentClassification)
class DocumentClassificationAdmin(GenericAdmin[DocumentClassification]):
    """
    Document Classification Admin
    """

    model: type[DocumentClassification] = DocumentClassification
    list_display = (
        "name",
        "description",
    )
    search_fields = ("name",)


@admin.register(ChargeType)
class ChargeTypeAdmin(GenericAdmin[ChargeType]):
    """
    Charge Type Admin
    """

    model: type[ChargeType] = ChargeType
    list_display = (
        "name",
        "description",
    )
    search_fields = ("name",)


@admin.register(AccessorialCharge)
class AccessorialChargeAdmin(GenericAdmin[AccessorialCharge]):
    """
    Accessorial Charge Admin
    """

    model: type[AccessorialCharge] = AccessorialCharge
    list_display = (
        "code",
        "charge_amount",
        "method",
    )
    search_fields = ("code",)
