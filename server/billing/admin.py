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

from django.contrib import admin
from django.http import HttpRequest

from billing.models import (
    AccessorialCharge,
    BillingControl,
    BillingException,
    BillingHistory,
    BillingLogEntry,
    BillingQueue,
    ChargeType,
    DocumentClassification,
    InvoicePaymentDetail,
)
from utils.admin import GenericAdmin


@admin.register(BillingQueue)
class BillingQueueAdmin(GenericAdmin[BillingQueue]):
    """
    Billing Queue Admin
    """

    model: type[BillingQueue] = BillingQueue
    list_display = (
        "shipment",
        "bol_number",
        "invoice_number",
        "bill_type",
    )
    search_fields = ("invoice_number", "bol_number")


@admin.register(BillingHistory)
class BillingHistoryAdmin(GenericAdmin[BillingHistory]):
    """
    Billing History Admin
    """

    model: type[BillingHistory] = BillingHistory
    list_display = (
        "shipment",
        "bol_number",
        "invoice_number",
    )
    search_fields = ("invoice_number", "shipment", "bol_number")

    def has_change_permission(
        self, request: HttpRequest, obj: BillingHistory | None = None
    ) -> bool:
        """Has permission to change.

        Args:
            request (HttpRequest): Request object from the view function that called this method (if any).
            obj (BillingHistory | None): Object to be deleted (if any).

        Returns:
            bool: True if the user has permission to delete the given object, False otherwise.
        """
        return False

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


@admin.register(BillingLogEntry)
class BillingLogEntry(GenericAdmin[BillingLogEntry]):
    """
    Billing Transfer Log Admin
    """

    model: type[BillingLogEntry] = BillingLogEntry
    list_display = (
        "customer",
        "created",
        "action",
    )
    search_fields = (
        "customer__name",
        "task_id",
        "created",
    )
    readonly_fields = (
        "customer",
        "task_id",
        "created",
    )

    def has_delete_permission(
        self, request: HttpRequest, obj: BillingLogEntry | None = None
    ) -> bool:
        """Has permission to delete.

        Args:
            request (HttpRequest): Request object from the view function that called this method (if any).
            obj (BillingLogEntry | None): Object to be deleted (if any).

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
        "shipment",
        "exception_type",
    )
    search_fields = (
        "exception_type",
        "shipment",
    )


@admin.register(BillingControl)
class BillingControlAdmin(GenericAdmin[BillingControl]):
    """
    Billing Control Admin
    """

    model: type[BillingControl] = BillingControl
    list_display = ("organization", "auto_bill_shipment")
    search_fields = ("organization", "auto_bill_shipment")


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


@admin.register(InvoicePaymentDetail)
class InvoicePaymentDetailAdmin(GenericAdmin[InvoicePaymentDetail]):
    """
    Invoice Payment Detail Admin
    """

    model: type[InvoicePaymentDetail] = InvoicePaymentDetail
    list_display = (
        "invoice",
        "payment_date",
        "payment_amount",
        "payment_method",
    )
    search_fields = ("invoice",)
