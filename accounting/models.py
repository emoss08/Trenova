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

import textwrap
from typing import Any, final

from django.core import validators
from django.core.exceptions import ValidationError
from django.db import models
from django.urls import reverse
from django.utils.translation import gettext_lazy as _

from core.models import GenericModel


class GeneralLedgerAccount(GenericModel):
    """
    Stores general ledger account information for related :model:`organization.Organization`.
    """

    @final
    class AccountTypeChoices(models.TextChoices):
        """
        General Ledger account types.
        """

        ASSET = "ASSET", _("Asset")
        LIABILITY = "LIABILITY", _("Liability")
        EQUITY = "EQUITY", _("Equity")
        REVENUE = "REVENUE", _("Revenue")
        EXPENSE = "EXPENSE", _("Expense")

    @final
    class CashFlowTypeChoices(models.TextChoices):
        """
        General Ledger account cash flow types.
        """

        OPERATING = "OPERATING", _("Operating")
        INVESTING = "INVESTING", _("Investing")
        FINANCING = "FINANCING", _("Financing")

    @final
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

    @final
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

    is_active = models.BooleanField(
        _("Active"),
        default=True,
        help_text=_("Designates whether this account is active."),
    )
    account_number = models.CharField(
        _("Account Number"),
        max_length=20,
        unique=True,
        help_text=_("The account number of the general ledger account."),
        validators=[
            validators.RegexValidator(
                regex=r"^[0-9]{4}-[0-9]{4}-[0-9]{4}-[0-9]{4}$",
                message=_("Account number must be in the format 0000-0000-0000-0000."),
            )
        ],
    )
    description = models.CharField(
        _("Description"),
        max_length=100,
        help_text=_("The description of the general ledger account."),
    )
    account_type = models.CharField(
        _("Account Type"),
        max_length=9,
        choices=AccountTypeChoices.choices,
        help_text=_("The type of the general ledger account."),
    )
    cash_flow_type = models.CharField(
        _("Cash Flow Type"),
        max_length=9,
        choices=CashFlowTypeChoices.choices,
        help_text=_("The cash flow type of the general ledger account."),
        blank=True,
    )
    account_sub_type = models.CharField(
        _("Account Sub Type"),
        max_length=20,
        choices=AccountSubTypeChoices.choices,
        help_text=_("The sub type of the general ledger account."),
        blank=True,
    )
    account_classification = models.CharField(
        _("Account Classification"),
        max_length=20,
        choices=AccountClassificationChoices.choices,
        help_text=_("The classification of the general ledger account."),
        blank=True,
    )

    class Meta:
        verbose_name = _("General Ledger Account")
        verbose_name_plural = _("General Ledger Accounts")
        ordering = ["account_number"]

    def __str__(self) -> str:
        """GeneralLedgerAccount string representation

        Returns:
            str: GeneralLedgerAccount string representation
        """
        return textwrap.wrap(self.account_number, 20)[0]

    def get_absolute_url(self) -> str:
        """GeneralLedgerAccount absolute url

        Returns:
            str: GeneralLedgerAccount absolute url
        """
        return reverse(
            "accounting:general_ledger_account_detail", kwargs={"pk": self.pk}
        )


class RevenueCode(GenericModel):
    """
    Stores revenue code information for related :model:`organization.Organization`.
    """

    code = models.CharField(
        _("Code"),
        max_length=4,
        unique=True,
        help_text=_("The revenue code."),
    )
    description = models.CharField(
        _("Description"),
        max_length=100,
        help_text=_("The description of the revenue code."),
    )
    expense_account = models.ForeignKey(
        GeneralLedgerAccount,
        on_delete=models.CASCADE,
        related_name="revenue_code_expense_account",
        related_query_name="revenue_code_expense_accounts",
        help_text=_("The expense account associated with the revenue code."),
        verbose_name=_("Expense Account"),
    )
    revenue_account = models.ForeignKey(
        GeneralLedgerAccount,
        on_delete=models.CASCADE,
        related_name="revenue_code_revenue_account",
        related_query_name="revenue_code_revenue_accounts",
        help_text=_("The revenue account associated with the revenue code."),
        verbose_name=_("Revenue Account"),
    )

    class Meta:
        verbose_name = _("Revenue Code")
        verbose_name_plural = _("Revenue Codes")
        ordering: list[str] = ["code"]

    def __str__(self) -> str:
        """RevenueCode string representation

        Returns:
            str: RevenueCode string representation
        """
        return textwrap.wrap(self.code, 4)[0]

    def clean(self) -> None:
        """RevenueCode model validation

        Returns:
            None
        """
        if (
                self.expense_account.account_type !=
                GeneralLedgerAccount.AccountTypeChoices.EXPENSE
        ):
            raise ValidationError(
                {"expense_account": _("Entered account is not an expense account.")}
            )
        if (
                self.revenue_account.account_type !=
                GeneralLedgerAccount.AccountTypeChoices.REVENUE
        ):
            raise ValidationError(
                {"revenue_account": _("Entered account is not a revenue account.")}
            )
        super().clean()

    def save(self, *args: Any, **kwargs: Any) -> None:
        """RevenueCode save method

        Args:
            *args (Any): Variable length argument list.
            **kwargs (Any): Arbitrary keyword arguments

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
        return reverse("accounting:revenue_code_detail", kwargs={"pk": self.pk})
