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

from __future__ import annotations

import textwrap
import typing
import uuid

from django.core import validators
from django.core.exceptions import ValidationError
from django.db import models
from django.db.models.functions import Lower
from django.urls import reverse
from django.utils import timezone
from django.utils.translation import gettext_lazy as _

from utils.models import ChoiceField, GenericModel, PrimaryStatusChoices


def attachment_upload_to(instance: GeneralLedgerAccount, filename: str) -> str:
    """Generate the upload path for the attachment field.

    Args:
        instance (GeneralLedgerAccount): The GeneralLedgerAccount instance.
        filename (str): The filename of the attachment.

    Returns:
        str: The upload path for the attachment field.
    """
    return f"{instance.organization_id}/gl_accounts/attachments/{filename}"


@typing.final
class TransactionTypeChoices(models.TextChoices):
    """
    Transaction types.
    """

    DEBIT = "REVENUE", _("Revenue")
    CREDIT = "EXPENSE", _("Expense")


@typing.final
class TransactionStatusChoices(models.TextChoices):
    """
    Transaction statuses.
    """

    PENDING = "PENDING", _("Pending")
    PENDING_RECON = "PENDING_RECON", _("Pending Reconciliation")
    COMPLETED = "COMPLETED", _("Completed")
    FAILED = "FAILED", _("Failed")


@typing.final
class ThresholdActionChoices(models.TextChoices):
    """
    ThresholdAction actions.
    """

    HALT = "HALT", _("Halt")
    WARN = "WARN", _("Warn")


class AccountingControl(GenericModel):
    """
    Stores the accounting control information for a related :model:`organization.Organization`.

    The AccountingControl model stores accounting configurations and controls for a related organization.
    It allows organizations to customize how their financial transactions are handled, including
    automation preferences and accounting constraints.
    """

    @typing.final
    class AutomaticJournalEntryChoices(models.TextChoices):
        """
        Represents the criteria choices for auto-creating journal entries.
        """

        ON_SHIPMENT_BILL = "ON_SHIPMENT_BILL", _(
            "Auto create entry when shipment is billed"
        )
        ON_RECEIPT_OF_PAYMENT = "ON_RECEIPT_OF_PAYMENT", _(
            "Auto create entry on receipt of payment"
        )
        ON_EXPENSE_RECOGNITION = "ON_EXPENSE_RECOGNITION", _(
            "Auto create entry when an expense is recognized"
        )

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
    )
    organization = models.OneToOneField(
        "organization.Organization",
        on_delete=models.CASCADE,
        verbose_name=_("Organization"),
        related_name="accounting_control",
    )
    auto_create_journal_entries = models.BooleanField(
        verbose_name=_("Automatically Create Journal Entries"),
        default=False,
        help_text=_(
            "Whether to automatically create journal entries based on certain triggers."
        ),
    )
    journal_entry_criteria = models.CharField(
        verbose_name=_("Journal Entry Criteria"),
        max_length=50,
        choices=AutomaticJournalEntryChoices.choices,
        default=AutomaticJournalEntryChoices.ON_SHIPMENT_BILL,
        help_text=_(
            "Define a criteria on when automatic journal entries are to be created."
        ),
        blank=True,
    )
    restrict_manual_journal_entries = models.BooleanField(
        verbose_name=_("Restrict Manual Journal Entries"),
        default=False,
        help_text=_(
            "If set to True, users will not be able to manually create journal entries without specific permissions."
        ),
    )
    require_journal_entry_approval = models.BooleanField(
        verbose_name=_("Require Approval for Journal Entries"),
        default=True,
        help_text=_(
            "If set to True, all created journal entries will need to be reviewed and approved by authorized "
            "personnel before being finalized."
        ),
    )
    default_revenue_account = models.ForeignKey(
        "GeneralLedgerAccount",
        on_delete=models.SET_NULL,
        related_name="default_revenue_for_accounting_control",
        null=True,
        blank=True,
        help_text=_("Default revenue account if no specific RevenueCode is provided."),
        verbose_name=_("Default Revenue Account"),
    )
    default_expense_account = models.ForeignKey(
        "GeneralLedgerAccount",
        on_delete=models.SET_NULL,
        related_name="default_expense_for_accounting_control",
        null=True,
        blank=True,
        help_text=_("Default expense account if no specific RevenueCode is provided."),
        verbose_name=_("Default Expense Account"),
    )
    enable_reconciliation_notifications = models.BooleanField(
        verbose_name=_("Enable Reconciliation Notifications"),
        default=True,
        help_text=_(
            "Send notifications when shipments are added to the reconciliation queue."
        ),
    )
    reconciliation_notification_recipients = models.ManyToManyField(
        "accounts.User",
        blank=True,
        verbose_name=_("Reconciliation Notification Recipients"),
        help_text=_(
            "Users who will receive notifications about reconciliation tasks. Leave empty for default "
            "recipients."
        ),
        related_name="reconciliation_notifications",
    )
    reconciliation_threshold = models.PositiveIntegerField(
        verbose_name=_("Reconciliation Threshold"),
        default=50,
        help_text=_(
            "Threshold for pending reconciliation tasks. If exceeded, can trigger warnings or halt certain processes."
        ),
    )
    reconciliation_threshold_action = ChoiceField(
        _("Reconciliation Threshold Action"),
        choices=ThresholdActionChoices.choices,
        default=ThresholdActionChoices.WARN,
        help_text=_("Action to be taken when the reconciliation threshold is reached."),
    )

    # 3. Identifying critical processes that shouldn't proceed if reconciliation tasks are pending.
    halt_on_pending_reconciliation = models.BooleanField(
        _("Halt on Pending Reconciliation"),
        default=False,
        help_text=_(
            "Halt critical processes if there are pending reconciliation tasks above the threshold."
        ),
    )
    critical_processes = models.TextField(
        _("Critical Processes"),
        blank=True,
        help_text=_(
            "List of critical processes that shouldn't proceed if pending reconciliation tasks are above the "
            "threshold. Define clear identifiers or names for each process."
        ),
    )

    class Meta:
        """
        Metaclass for AccountingControl
        """

        verbose_name = _("Accounting Control")
        verbose_name_plural = _("Accounting Controls")
        db_table = "accounting_control"
        permissions = [
            ("accounting.use_accounting_client", "Can use the accounting client"),
            ("accounting.approve_journal_entry", "Can approve journal entries"),
        ]
        db_table_comment = (
            "Stores the accounting control information for a related organization."
        )

    def __str__(self) -> str:
        """AccountingControl string representation

        Returns:
            str: AccountingControl string representation
        """
        return textwrap.shorten(
            f"Accounting Control for {self.organization}", width=40, placeholder="..."
        )

    def get_absolute_url(self) -> str:
        """AccountingControl absolute url

        Returns:
            str: AccountingControl absolute url
        """
        return reverse("accounting-control-detail", kwargs={"pk": self.pk})

    def clean(self) -> None:
        """RevenueCode model validation

        Returns:
            None
        """

        super().clean()

        errors = {}

        if (
            self.default_expense_account
            and self.default_expense_account.account_type
            != GeneralLedgerAccount.AccountTypeChoices.EXPENSE
        ):
            errors["default_expense_account"] = _(
                f"Entered account is a {self.default_expense_account.account_type} account, not a expense account. "
                f"Please try again."
            )

        if (
            self.default_revenue_account
            and self.default_revenue_account.account_type
            != GeneralLedgerAccount.AccountTypeChoices.REVENUE
        ):
            errors["default_revenue_account"] = _(
                f"Entered account is a {self.default_revenue_account.account_type} account, not a revenue account. "
                f"Please try again."
            )

        if errors:
            raise ValidationError(errors)


class GeneralLedgerAccount(GenericModel):
    """
    Stores general ledger account information for related :model:`organization.Organization`.
    """

    @typing.final
    class AccountTypeChoices(models.TextChoices):
        """
        General Ledger account types.
        """

        ASSET = "ASSET", _("Asset")
        LIABILITY = "LIABILITY", _("Liability")
        EQUITY = "EQUITY", _("Equity")
        REVENUE = "REVENUE", _("Revenue")
        EXPENSE = "EXPENSE", _("Expense")

    @typing.final
    class CashFlowTypeChoices(models.TextChoices):
        """
        General Ledger account cash flow types.
        """

        OPERATING = "OPERATING", _("Operating")
        INVESTING = "INVESTING", _("Investing")
        FINANCING = "FINANCING", _("Financing")

    @typing.final
    class AccountSubTypeChoices(models.TextChoices):
        """
        General Ledger account subtypes.
        """

        CURRENT_ASSET = "CURRENT_ASSET", _("Current Asset")
        FIXED_ASSET = "FIXED_ASSET", _("Fixed Asset")
        OTHER_ASSET = "OTHER_ASSET", _("Other Asset")
        CURRENT_LIABILITY = "CURRENT_LIABILITY", _("Current Liability")
        LONG_TERM_LIABILITY = "LONG_TERM_LIABILITY", _("Long Term Liability")
        EQUITY = "EQUITY", _("Equity")
        REVENUE = "REVENUE", _("Revenue")
        COST_OF_GOODS_SOLD = "COST_OF_GOODS_SOLD", _("Cost of Goods Sold")
        EXPENSE = "EXPENSE", _("Expense")
        OTHER_INCOME = "OTHER_INCOME", _("Other Income")
        OTHER_EXPENSE = "OTHER_EXPENSE", _("Other Expense")

    @typing.final
    class AccountClassificationChoices(models.TextChoices):
        """
        General Ledger account classifications.
        """

        BANK = "BANK", _("Bank")
        CASH = "CASH", _("Cash")
        ACCOUNTS_RECEIVABLE = "ACCOUNTS_RECEIVABLE", _("Accounts Receivable")
        ACCOUNTS_PAYABLE = "ACCOUNTS_PAYABLE", _("Accounts Payable")
        INVENTORY = "INVENTORY", _("Inventory")
        OTHER_CURRENT_ASSET = "OTHER_CURRENT_ASSET", _("Other Current Asset")
        FIXED_ASSET = "FIXED_ASSET", _("Fixed Asset")

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
    )
    status = ChoiceField(
        _("Status"),
        choices=PrimaryStatusChoices.choices,
        help_text=_("Status of the General Ledger Account."),
        default=PrimaryStatusChoices.ACTIVE,
    )
    account_number = models.CharField(
        _("Account Number"),
        max_length=7,
        help_text=_("The account number of the General Ledger Account."),
        validators=[
            validators.RegexValidator(
                regex=r"^[0-9]{4}-[0-9]{2}$",
                message=_("Account number must be in the format XXXX-XX."),
            )
        ],
    )
    description = models.CharField(
        _("Description"),
        max_length=100,
        help_text=_("The description of the General Ledger Account."),
    )
    account_type = ChoiceField(
        _("Account Type"),
        choices=AccountTypeChoices.choices,
        help_text=_("The type of the General Ledger Account."),
    )
    cash_flow_type = ChoiceField(
        _("Cash Flow Type"),
        choices=CashFlowTypeChoices.choices,
        help_text=_("The cash flow type of the General Ledger Account."),
        blank=True,
    )
    account_sub_type = ChoiceField(
        _("Account Sub Type"),
        choices=AccountSubTypeChoices.choices,
        help_text=_("The sub type of the General Ledger Account."),
        blank=True,
    )
    account_classification = ChoiceField(
        _("Account Classification"),
        choices=AccountClassificationChoices.choices,
        help_text=_("The classification of the General Ledger Account."),
        blank=True,
    )
    balance = models.DecimalField(
        verbose_name=_("Balance"),
        max_digits=20,
        decimal_places=2,
        default=0,
        help_text=_("Current balance of the account."),
    )
    opening_balance = models.DecimalField(
        verbose_name=_("Opening Balance"),
        max_digits=20,
        decimal_places=2,
        default=0,
        help_text=_("Opening balance of the account."),
    )
    closing_balance = models.DecimalField(
        verbose_name=_("Closing Balance"),
        max_digits=20,
        decimal_places=2,
        default=0,
        help_text=_("Closing balance of the account."),
    )
    parent_account = models.ForeignKey(
        "self",
        on_delete=models.SET_NULL,
        blank=True,
        null=True,
        related_name="child_accounts",
        help_text=_("Parent account for hierarchical accounting."),
        verbose_name=_("Parent Account"),
    )
    is_reconciled = models.BooleanField(
        verbose_name=_("Is Reconciled?"),
        default=False,
        help_text=_("Indicates if the account is reconciled."),
    )
    date_opened = models.DateField(
        verbose_name=_("Date Opened"),
        auto_now_add=True,
        help_text=_("Date when the account was opened."),
    )
    date_closed = models.DateField(
        verbose_name=_("Date Closed"),
        blank=True,
        null=True,
        help_text=_("Date when the account was closed."),
    )
    notes = models.TextField(
        verbose_name=_("Notes"),
        blank=True,
        help_text=_("Additional notes or comments for the account."),
    )
    owner = models.ForeignKey(
        "accounts.User",
        on_delete=models.SET_NULL,
        blank=True,
        null=True,
        help_text=_("User responsible for the account."),
        verbose_name=_("Owner"),
    )
    is_tax_relevant = models.BooleanField(
        verbose_name=_("Is Tax Relevant?"),
        default=False,
        help_text=_("Indicates if the account is relevant for tax calculations."),
    )
    attachment = models.FileField(
        verbose_name=_("Attachment"),
        upload_to=attachment_upload_to,
        blank=True,
        null=True,
        help_text=_("Attach relevant documents or receipts."),
    )
    interest_rate = models.DecimalField(
        verbose_name=_("Interest Rate"),
        max_digits=5,
        decimal_places=2,
        blank=True,
        null=True,
        help_text=_("Interest rate associated with the account (if applicable)."),
    )
    tags = models.ManyToManyField(
        "Tag",
        blank=True,
        related_name="ledger_accounts",
        help_text=_("Tags or labels associated with the account."),
        verbose_name=_("Tags"),
    )

    class Meta:
        """
        Metaclass for GeneralLedgerAccount Model
        """

        verbose_name = _("General Ledger Account")
        verbose_name_plural = _("General Ledger Accounts")
        ordering = ["account_number"]
        db_table = "general_ledger_account"
        constraints = [
            models.UniqueConstraint(
                Lower("account_number"),
                "organization",
                name="unique_gl_account_organization",
            )
        ]

    def __str__(self) -> str:
        """GeneralLedgerAccount string representation

        Returns:
            str: GeneralLedgerAccount string representation
        """
        return textwrap.shorten(
            f"{self.account_type} - {self.account_number}", width=40, placeholder="..."
        )

    def get_absolute_url(self) -> str:
        """GeneralLedgerAccount absolute url

        Returns:
            str: GeneralLedgerAccount absolute url
        """
        return reverse("gl-accounts-detail", kwargs={"pk": self.pk})


class Tag(GenericModel):
    """
    Represents a tag that can be associated with a General Ledger Account.
    """

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
    )
    name = models.CharField(
        verbose_name=_("Name"), max_length=100, help_text=_("Name of the tag.")
    )
    description = models.TextField(
        verbose_name=_("Description"),
        blank=True,
        help_text=_("Optional description for the tag."),
    )

    class Meta:
        verbose_name = _("Tag")
        verbose_name_plural = _("Tags")
        ordering = ["name"]
        db_table = "tags"
        constraints = [
            models.UniqueConstraint(
                Lower("name"),
                "organization",
                name="unique_tag_organization",
            )
        ]

    def __str__(self) -> str:
        return textwrap.shorten(
            f"{self.name}, {self.description}", width=40, placeholder="..."
        )

    def get_absolute_url(self) -> str:
        """Tags absolute url

        Returns:
            str: Tags absolute url
        """
        return reverse("tags-detail", kwargs={"pk": self.pk})


class RevenueCode(GenericModel):
    """
    Stores revenue code information for related :model:`organization.Organization`.
    """

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
    )
    code = models.CharField(
        _("Code"),
        max_length=4,
        help_text=_("The revenue code."),
    )
    description = models.CharField(
        _("Description"),
        max_length=100,
        help_text=_("The description of the revenue code."),
    )
    expense_account = models.ForeignKey(
        GeneralLedgerAccount,
        on_delete=models.RESTRICT,
        related_name="revenue_code_expense_account",
        related_query_name="revenue_code_expense_accounts",
        help_text=_("The expense account associated with the revenue code."),
        verbose_name=_("Expense Account"),
        blank=True,
        null=True,
    )
    revenue_account = models.ForeignKey(
        GeneralLedgerAccount,
        on_delete=models.RESTRICT,
        related_name="revenue_code_revenue_account",
        related_query_name="revenue_code_revenue_accounts",
        help_text=_("The revenue account associated with the revenue code."),
        verbose_name=_("Revenue Account"),
        blank=True,
        null=True,
    )

    class Meta:
        """
        Metaclass for the RevenueCode Model
        """

        verbose_name = _("Revenue Code")
        verbose_name_plural = _("Revenue Codes")
        ordering = ["code"]
        db_table = "revenue_code"
        constraints = [
            models.UniqueConstraint(
                Lower("code"),
                "organization",
                name="unique_revenue_code_organization",
            )
        ]

    def __str__(self) -> str:
        """RevenueCode string representation

        Returns:
            str: RevenueCode string representation
        """
        return textwrap.wrap(self.code, 4)[0]

    def save(self, *args: typing.Any, **kwargs: typing.Any) -> None:
        """RevenueCode save method

        Args:
            *args (typing.Any): Variable length argument list.
            **kwargs (typing.Any): Arbitrary keyword arguments

        Returns:
            None
        """
        self.full_clean()

        if self.code:
            self.code = self.code.upper()

        super().save(*args, **kwargs)

    def get_absolute_url(self) -> str:
        """RevenueCode absolute url

        Returns:
            str: RevenueCode absolute url
        """
        return reverse("revenue_code_detail", kwargs={"pk": self.pk})

    def clean(self) -> None:
        """RevenueCode model validation

        Returns:
            None
        """

        super().clean()

        errors = {}

        if (
            self.expense_account
            and self.expense_account.account_type
            != GeneralLedgerAccount.AccountTypeChoices.EXPENSE
        ):
            errors["expense_account"] = _(
                f"Entered account is a {self.expense_account.account_type} account, not a expense account. Please try again."
            )

        if (
            self.revenue_account
            and self.revenue_account.account_type
            != GeneralLedgerAccount.AccountTypeChoices.REVENUE
        ):
            errors["revenue_account"] = _(
                f"Entered account is a {self.revenue_account.account_type} account, not a revenue account. Please try again."
            )

        if errors:
            raise ValidationError(errors)

    def update_revenue_code(self, **kwargs: typing.Any) -> None:
        """Update the revenue code.

        Args:
            **kwargs: Keyword arguments

        Examples:
            >>> revenue_code = RevenueCode.objects.get(pk=uuid.uuid4())
            >>> revenue_code.update_revenue_code(code="1234", description="New Description")

        Returns:
            None: This function does not return anything.
        """

        for key, value in kwargs.items():
            setattr(self, key, value)
        self.save()


class DivisionCode(GenericModel):
    """
    Stores division code information for related :model:`organization.Organization`.
    """

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
    )
    status = ChoiceField(
        _("Status"),
        choices=PrimaryStatusChoices.choices,
        help_text=_("Status of the division code."),
        default=PrimaryStatusChoices.ACTIVE,
    )
    code = models.CharField(
        _("Code"),
        max_length=4,
        help_text=_("The division code."),
    )
    description = models.CharField(
        _("Description"),
        max_length=100,
        help_text=_("The description of the division code."),
    )
    cash_account = models.ForeignKey(
        GeneralLedgerAccount,
        on_delete=models.RESTRICT,
        related_name="division_code_cash_account",
        help_text=_("The cash account associated with the division code."),
        verbose_name=_("Cash Account"),
        blank=True,
        null=True,
    )
    ap_account = models.ForeignKey(
        GeneralLedgerAccount,
        on_delete=models.RESTRICT,
        related_name="division_code_ap_account",
        help_text=_("The accounts payable account associated with the division code."),
        verbose_name=_("Accounts Payable Account"),
        blank=True,
        null=True,
    )
    expense_account = models.ForeignKey(
        GeneralLedgerAccount,
        on_delete=models.RESTRICT,
        related_name="division_code_expense_account",
        help_text=_("The expense account associated with the division code."),
        verbose_name=_("Expense Account"),
        blank=True,
        null=True,
    )

    class Meta:
        """
        Metaclass for DivisionCode Model
        """

        verbose_name = _("Division Code")
        verbose_name_plural = _("Division Codes")
        ordering = ["code"]
        db_table = "division_code"
        db_table_comment = "Stores division code information for related organization."
        constraints = [
            models.UniqueConstraint(
                Lower("code"),
                "organization",
                name="unique_division_code_organization",
            )
        ]

    def __str__(self) -> str:
        """DivisionCode string representation

        Returns:
            str: DivisionCode string representation
        """
        return textwrap.wrap(self.code, 4)[0]

    def get_absolute_url(self) -> str:
        """DivisionCode absolute url

        Returns:
            str: DivisionCode absolute url
        """
        return reverse("division-codes-detail", kwargs={"pk": self.pk})

    def clean(self) -> None:
        """DivisionCode model validation method

        Returns:
            None: This function does not return anything.

        Raises:
            ValidationError: If the cash account is not a cash account.
            ValidationError: If the expense account is not an expense account.
            ValidationError: If the ap account is not an accounts payable account.
        """

        errors = {}

        super().clean()

        if (
            self.cash_account
            and self.cash_account.account_classification
            != GeneralLedgerAccount.AccountClassificationChoices.CASH
        ):
            errors["cash_account"] = _(
                "Entered account is not an cash account. Please try again."
            )

        if (
            self.expense_account
            and self.expense_account.account_type
            != GeneralLedgerAccount.AccountTypeChoices.EXPENSE
        ):
            errors["expense_account"] = _(
                "Entered account is not an expense account. Please try again."
            )

        if (
            self.ap_account
            and self.ap_account.account_classification
            != GeneralLedgerAccount.AccountClassificationChoices.ACCOUNTS_PAYABLE
        ):
            errors["ap_account"] = _(
                "Entered account is not an accounts payable account. Please try again."
            )

        if errors:
            raise ValidationError(errors)


class FinancialTransaction(GenericModel):
    """
    Model representing a financial transaction in the ledger.
    """

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
    )
    transaction_number = models.CharField(
        verbose_name=_("Transaction Number"),
        max_length=50,
        blank=True,
        editable=False,
        help_text=_("Transaction number associated with the transaction"),
    )
    amount = models.DecimalField(
        verbose_name=_("Amount"),
        max_digits=15,
        decimal_places=2,
        help_text=_("Amount of the transaction"),
    )
    transaction_type = ChoiceField(
        verbose_name=_("Transaction Type"),
        max_length=10,
        choices=TransactionTypeChoices.choices,
        help_text=_("Type of the transaction"),
    )
    date_created = models.DateTimeField(
        verbose_name=_("Date Created"),
        auto_now_add=True,
        help_text=_("Date and time of the transaction"),
    )
    ledger_account = models.ForeignKey(
        GeneralLedgerAccount,
        on_delete=models.CASCADE,
        related_name="transactions",
        related_query_name="transaction",
        help_text=_("General Ledger Account associated with the transaction"),
        verbose_name=_("General Ledger Account"),
    )
    shipment = models.ForeignKey(
        "shipment.Shipment",
        on_delete=models.SET_NULL,
        null=True,
        blank=True,
        related_name="financial_transactions",
        related_query_name="financial_transaction",
        help_text=_("Shipment associated with the transaction"),
        verbose_name=_("Shipment"),
    )
    status = ChoiceField(
        choices=TransactionStatusChoices.choices,
        default=TransactionStatusChoices.PENDING,
        help_text=_("Status of the transaction"),
        verbose_name=_("Status"),
    )
    created_by = models.ForeignKey(
        "accounts.User",
        on_delete=models.SET_NULL,
        null=True,
        blank=True,
        related_name="created_transactions",
        related_query_name="created_transaction",
        help_text=_("User who created/triggered this transaction"),
        verbose_name=_("Created By"),
    )
    description = models.TextField(
        verbose_name=_("Description"),
        blank=True,
        help_text=_("Description or note for the transaction"),
    )
    external_reference = models.CharField(
        verbose_name=_("External Reference"),
        max_length=100,
        blank=True,
        help_text=_(
            "External reference or invoice number associated with the transaction"
        ),
    )

    class Meta:
        """
        Metaclass for FinancialTransaction Model
        """

        verbose_name = _("Financial Transaction")
        verbose_name_plural = _("Financial Transactions")
        ordering = ["transaction_type", "amount"]
        db_table = "financial_transaction"
        db_table_comment = "Model representing a financial transaction in the ledger."
        constraints = [
            models.UniqueConstraint(
                Lower("transaction_number"),
                "organization",
                name="unique_transaction_number_organization",
            )
        ]

    def __str__(self) -> str:
        """FinancialTransaction string representation

        Returns:
            str: FinancialTransaction string representation
        """
        return textwrap.shorten(
            f"{self.transaction_type} - {self.transaction_number}",
            width=40,
            placeholder="...",
        )

    def save(self, *args: typing.Any, **kwargs: typing.Any) -> None:
        """FinancialTransaction save method override

        Args:
            *args (Any): Arguments
            **kwargs: Keyword Arguments

        Returns:
            None: this function does not return anything.
        """
        if not self.transaction_number:
            self.generate_transaction_number()

        super().save(*args, **kwargs)

    def get_absolute_url(self) -> str:
        """FinancialTransaction absolute url

        Returns:
            str: FinancialTransaction absolute url
        """
        return reverse("financial-transaction-detail", kwargs={"pk": self.pk})

    def generate_transaction_number(self) -> None:
        current_date = timezone.now().date().strftime("%Y%m%d")

        if last_transaction := (
            self.__class__.objects.filter(transaction_number__startswith=current_date)
            .order_by("-transaction_number")
            .first()
        ):
            last_seq_num = int(last_transaction.transaction_number.split("-")[-1])
            new_seq_num = str(last_seq_num + 1).zfill(3)
        else:
            new_seq_num = "001"

        self.transaction_number = f"{current_date}-{new_seq_num}"


class ReconciliationQueue(GenericModel):
    """Stores records for shipments that require reconciliation due to missing or inconsistent financial details.

    Attributes:
        id (UUIDField): Primary key and default value is a randomly generated UUID. Non-editable and unique.
        shipment (ForeignKey): ForeignKey to the Shipment model with CASCADE on delete. Represents the shipment that
            needs reconciliation.
        reason (TextField): A textual description of why this shipment requires reconciliation.
        date_added (DateTimeField): The timestamp when the shipment was added to the reconciliation queue.
        resolved (BooleanField): Indicates if the reconciliation for this record has been resolved.
        resolved_by (ForeignKey): ForeignKey to the User model with SET_NULL on delete. Represents the user who resolved
            the reconciliation.
        date_resolved (DateTimeField): The timestamp when the reconciliation was resolved.
        notes (TextField): Additional notes or comments about the reconciliation. Useful for logging any manual
            adjustments or decisions made during the reconciliation process.
    """

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
        verbose_name=_("Record ID"),
    )
    shipment = models.ForeignKey(
        "shipment.Shipment",
        on_delete=models.RESTRICT,
        verbose_name=_("Related Shipment"),
        related_name="reconciliation_records",
    )
    reason = models.TextField(
        verbose_name=_("Reconciliation Reason"),
        help_text=_("Reason for reconciliation being required."),
    )
    date_added = models.DateTimeField(
        verbose_name=_("Date Added to Queue"),
        auto_now_add=True,
        help_text=_(
            "Timestamp when the shipment was added to the reconciliation queue."
        ),
    )
    financial_transaction = models.ForeignKey(
        FinancialTransaction,
        on_delete=models.SET_NULL,
        blank=True,
        null=True,
        verbose_name=_("Related Financial Transaction"),
        related_name="reconciliation_records",
        help_text=_(
            "The specific financial transaction that triggered this reconciliation record."
        ),
    )
    resolved = models.BooleanField(
        verbose_name=_("Is Resolved?"),
        default=False,
        help_text=_("Whether the reconciliation for this record has been resolved."),
    )
    resolved_by = models.ForeignKey(
        "accounts.User",
        on_delete=models.SET_NULL,
        verbose_name=_("User Who Resolved"),
        blank=True,
        null=True,
        help_text=_("User who resolved the reconciliation."),
    )
    date_resolved = models.DateTimeField(
        verbose_name=_("Date of Resolution"),
        blank=True,
        null=True,
        help_text=_("Timestamp when the reconciliation was resolved."),
    )
    notes = models.TextField(
        verbose_name=_("Additional Notes"),
        blank=True,
        help_text=_("Additional notes or comments about the reconciliation process."),
    )

    class Meta:
        """
        Metaclass for ReconciliationQueue
        """

        verbose_name = _("Reconciliation Record")
        verbose_name_plural = _("Reconciliation Records")
        db_table = "reconciliation_queue"
        db_table_comment = (
            "Stores records for shipments that require reconciliation due to missing or inconsistent "
            "financial details."
        )
        permissions = [
            ("can_resolve_reconciliation", "Can resolve reconciliation issues"),
        ]

    def __str__(self):
        return textwrap.shorten(
            f"Reconciliation for Shipment {self.shipment.id} - {self.reason}",
            width=40,
            placeholder="...",
        )

    def get_absolute_url(self) -> str:
        """Reconciliation Record absolute url

        Returns:
            Absolute url for the reconciliation queue object. For example,
            `/reconciliation-records/edd1e612-cdd4-43d9-b3f3-bc099872088b/'
        """
        return reverse("reconciliation-records-detail", kwargs={"pk": self.pk})
