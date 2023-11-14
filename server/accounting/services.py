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
import logging
from smtplib import SMTPException

from django.core.mail import send_mail
from django.db import transaction

from accounting import models, selectors
from billing.selectors import get_invoice_payment_detail, get_shipment_bill_hist
from shipment.models import Shipment

logger = logging.getLogger(__name__)

REVENUE_CODE_STRING = "Revenue Code"


class TransactionService:
    """A service class to manage financial business transactions concerning the shipment of goods.

    The TransactionService class encompasses methods for creating transactions, checking data sufficiency,
    identifying missing data, handling reconciliation, among others, for the purpose of clean financial dealings in
    shipment-based businesses.

    All methods of this class are class methods, meaning they can be called directly on the class rather than an instance
    of the class.
    """

    @classmethod
    @transaction.atomic
    def create_transaction_from_shipment(cls, *, shipment: Shipment) -> None:
        """Create a transaction from a shipment. If there is insufficient data for the transaction, log it to
        reconciliation queue and handle the reconciliation process.

        Args:
            shipment (Shipment): Shipment object from which transaction will be created.

        Returns:
            None
        """
        control = selectors.get_accounting_control_by_org(
            organization=shipment.organization
        )

        revenue_account = (
            shipment.revenue_code.revenue_account
            if shipment.revenue_code
            else control.default_revenue_account
        )

        if not revenue_account:
            raise ValueError("No revenue account found for shipment.")

        f_transaction = cls._create_financial_transaction(
            shipment=shipment, revenue_account=revenue_account
        )

        if not cls._has_sufficient_data(shipment=shipment, control=control):
            missing_data = cls._identify_missing_data(shipment=shipment)
            cls._log_to_reconciliation_queue(
                shipment=shipment,
                f_transaction=f_transaction,
                missing_data=missing_data,
            )

        cls._handle_reconciliation(
            shipment=shipment, f_transaction=f_transaction, control=control
        )

    @classmethod
    def _create_financial_transaction(
        cls, *, shipment: Shipment, revenue_account: models.GeneralLedgerAccount
    ) -> models.FinancialTransaction:
        """Create a new financial transaction for a given shipment and revenue account.

        Args:
            shipment (Shipment): The shipment to create a transaction for.
            revenue_account (models.GeneralLedgerAccount): The revenue GL account related to the transaction.

        Returns:
            models.FinancialTransaction: The newly created FinancialTransaction object.
        """
        return models.FinancialTransaction.objects.create(
            business_unit=shipment.business_unit,
            transaction_type=models.TransactionTypeChoices.DEBIT,
            organization=shipment.organization,
            general_ledger_account=revenue_account,
            amount=round(shipment.sub_total, 2),
            status=models.TransactionStatusChoices.PENDING,
            shipment=shipment,
            created_by=shipment.entered_by,
        )

    @classmethod
    def _log_to_reconciliation_queue(
        cls,
        *,
        shipment: Shipment,
        f_transaction: models.FinancialTransaction,
        missing_data: str,
    ) -> None:
        """Log insufficient data for a transaction to reconciliation queue.

        Args:
            shipment (Shipment): The shipment object pertaining to the transaction.
            f_transaction (models.FinancialTransaction): The transaction that has insufficient data.
            missing_data (str): String summarizing the missing data.

        Returns:
            None: This function does not return anything.
        """
        models.ReconciliationQueue.objects.create(
            business_unit=shipment.business_unit,
            organization=shipment.organization,
            shipment=shipment,
            reason=f"Transaction {f_transaction.transaction_number}: Insufficient data for transaction"
            f" - {missing_data}",
            financial_transaction=f_transaction,
        )

    @classmethod
    def _has_sufficient_data(
        cls, *, shipment: Shipment, control: models.AccountingControl
    ) -> bool:
        """Check if a shipment has sufficient data for creating a transaction.

        Args:
            shipment (Shipment): The shipment object to check.
            control (models.AccountingControl): The AccountingControl object related to the shipment.

        Returns:
            bool: True if shipment has sufficient data, False otherwise."""
        default_revenue_account = control.default_revenue_account
        return bool(shipment.revenue_code or default_revenue_account)

    @classmethod
    def _identify_missing_data(cls, *, shipment: Shipment) -> str:
        """Identify the data missing from a shipment object for the purpose of transaction creation.

        Args:
            shipment (Shipment): The shipment object to inspect.

        Returns:
            str: String summarizing the missing data."""
        missing = []

        if not shipment.revenue_code:
            missing.append(REVENUE_CODE_STRING)

        return ", ".join(missing)

    @classmethod
    def _match_journal_entry_criteria(
        cls, *, shipment: Shipment, criteria: str
    ) -> bool:
        """Check if a shipment matches journal entry criteria.

        Args:
            shipment (Shipment): The shipment object to check.
            criteria (str): The journal entry criteria to match against.

        Returns:
            bool: True if shipment matches journal entry criteria, False otherwise."""
        if (
            criteria
            == models.AccountingControl.AutomaticJournalEntryChoices.ON_SHIPMENT_BILL
        ):
            return shipment.ready_to_bill
        return False

    @classmethod
    def _handle_reconciliation(
        cls,
        *,
        shipment: Shipment,
        f_transaction: models.FinancialTransaction,
        control: models.AccountingControl,
    ) -> None:
        """Handle the reconciliation process for a shipment and the related transaction, according to the given
        AccountingControl.

        Args:
            shipment (Shipment): The shipment corresponding to the transaction.
            f_transaction (models.FinancialTransaction): The transaction to reconcile.
            control (models.AccountingControl): The AccountingControl for the organization the shipment was made.

        Returns:
            None"""
        requires_reconciliation, reason = cls._shipment_requires_reconciliation(
            shipment=shipment, f_transaction=f_transaction, control=control
        )

        if requires_reconciliation and control.enable_reconciliation_notifications:
            cls._notify_reconciliation_required(
                shipment=shipment, reason=reason, control=control
            )
            models.ReconciliationQueue.objects.create(
                business_unit=shipment.business_unit,
                organization=shipment.organization,
                shipment=shipment,
                reason=reason,
                financial_transaction=f_transaction,
            )
            cls._log_reconciliation_event(shipment=shipment)

    @classmethod
    def _notify_reconciliation_required(
        cls, *, shipment: Shipment, reason: str, control: models.AccountingControl
    ) -> None:
        """Send a notification email to receipts when reconciliation is required for a shipment.

        Args:
            shipment (Shipment): Shipment object that requires reconciliation.
            reason (str): The reason why the shipment requires reconciliation.
            control (models.AccountingControl): The AccountingControl related to the shipment.

        Returns:
            None

        Raises:
            SMTPException: An error occurred when trying to send the email."""
        recipients = control.reconciliation_notification_recipients.all()
        if not recipients:
            return

        try:
            send_mail(
                subject=f"Shipment {shipment.pro_number} requires reconciliation.",
                message=f"Shipment {shipment.pro_number} requires reconciliation. Reason: {reason}",
                from_email="no-reply@monta.io",
                recipient_list=[user.email for user in recipients],
                fail_silently=False,
            )
        except SMTPException as email_error:
            logger.error(
                f"Error sending reconciliation notification email: {email_error}"
            )

    @classmethod
    def _log_reconciliation_event(cls, *, shipment: Shipment) -> None:
        """Log the event of reconciliation for a shipment.

        Args:
            shipment (Shipment): The shipment that reconciliation is logged for.

        Returns:
            None"""
        # Logic to log the event (could be saving to a database, sending to a monitoring system, etc.)
        print(f"Reconciliation logged for shipment {shipment.pro_number}.")

    @classmethod
    def _shipment_requires_reconciliation(
        cls,
        *,
        shipment: Shipment,
        f_transaction: models.FinancialTransaction,
        control: models.AccountingControl,
    ) -> tuple[bool, str]:
        """Determine whether a shipment requires reconciliation, and if true, provide the reason.

        Args:
            shipment (Shipment): The shipment to check.
            f_transaction (models.FinancialTransaction): The transaction related to the shipment.
            control (models.AccountingControl): The AccountingControl object related to the shipment.

        Returns:
            tuple[bool, str]: Pair of boolean indicating if reconciliation is required and a string
            giving the reason. If reconciliation is not required, the reason is an empty string.
        """
        if not shipment.revenue_code:
            return (
                True,
                f"Transaction {f_transaction.transaction_number}: Missing Revenue Code.",
            )

        billed_shipment = get_shipment_bill_hist(shipment=shipment)

        if billed_shipment is None:
            return (
                True,
                f"Transaction {f_transaction.transaction_number}: No billed shipment found for Shipment: "
                f"{shipment.pro_number}.",
            )

        if billed_shipment.total_amount != shipment.sub_total:
            return (
                True,
                f"Transaction {f_transaction.transaction_number}: Amount discrepancy detected for "
                f"Shipment: {shipment.pro_number}. Expected billed amount: {billed_shipment.total_amount},"
                f" but found: {shipment.sub_total}.",
            )

        invoice = get_invoice_payment_detail(invoice=billed_shipment)

        if (
            invoice is not None
            and billed_shipment.total_amount
            and invoice.payment_amount != billed_shipment.total_amount
        ):
            payment_diff = invoice.payment_amount - billed_shipment.total_amount
            return (
                True,
                (
                    f"Transaction {f_transaction.transaction_number}: Payment discrepancy detected for Shipment:"
                    f" {shipment.pro_number}. Expected payment: {billed_shipment.total_amount}, but"
                    f" received: {invoice.payment_amount}. Payment difference: {payment_diff}."
                ),
            )

        criteria = control.journal_entry_criteria
        if not cls._match_journal_entry_criteria(shipment=shipment, criteria=criteria):
            return True, (
                f"Transaction {f_transaction.transaction_number}: Journal Entry Criteria mismatch for"
                f" Shipment: {shipment.pro_number}. Shipment does not meet the '{criteria}' criteria set "
                f"in the Accounting Control."
            )

        return False, ""  # No reconciliation needed
