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

import textwrap
import uuid
from typing import Any, final

from django.core import validators
from django.core.exceptions import ValidationError
from django.db import models
from django.urls import reverse
from django.utils.translation import gettext_lazy as _

from utils.models import ChoiceField, GenericModel


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

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
    )
    is_active = models.BooleanField(
        _("Active"),
        default=True,
        help_text=_("Designates whether this account is active."),
    )
    account_number = models.CharField(
        _("Account Number"),
        max_length=20,
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
    account_type = ChoiceField(
        _("Account Type"),
        choices=AccountTypeChoices.choices,
        help_text=_("The type of the general ledger account."),
    )
    cash_flow_type = ChoiceField(
        _("Cash Flow Type"),
        choices=CashFlowTypeChoices.choices,
        help_text=_("The cash flow type of the general ledger account."),
        blank=True,
    )
    account_sub_type = ChoiceField(
        _("Account Sub Type"),
        choices=AccountSubTypeChoices.choices,
        help_text=_("The sub type of the general ledger account."),
        blank=True,
    )
    account_classification = ChoiceField(
        _("Account Classification"),
        choices=AccountClassificationChoices.choices,
        help_text=_("The classification of the general ledger account."),
        blank=True,
    )

    class Meta:
        """
        Metaclass for GeneralLedgerAccount Model
        """

        verbose_name = _("General Ledger Account")
        verbose_name_plural = _("General Ledger Accounts")
        ordering = ["account_number"]
        db_table = "general_ledger_account"

    def __str__(self) -> str:
        """GeneralLedgerAccount string representation

        Returns:
            str: GeneralLedgerAccount string representation
        """
        return textwrap.wrap(self.account_number, 20)[0]

    def clean(self) -> None:
        """General ledger account clean method

        Returns:
            None

        Raises:
            ValidationError: If account number is not unique
        """
        super().clean()

        if (
            GeneralLedgerAccount.objects.filter(
                account_number__exact=self.account_number
            )
            .exclude(id__exact=self.id)
            .exists()
        ):
            raise ValidationError(
                {
                    "account_number": _(
                        "An account with this account number already exists. Please try again."
                    )
                },
                code="invalid",
            )

    def get_absolute_url(self) -> str:
        """GeneralLedgerAccount absolute url

        Returns:
            str: GeneralLedgerAccount absolute url
        """
        return reverse("gl-accounts-detail", kwargs={"pk": self.pk})

    def update_gl_account(self, **kwargs):
        """Update the General Ledger account

        Args:
            **kwargs: Keyword arguments
        """

        for key, value in kwargs.items():
            setattr(self, key, value)
        self.save(**kwargs)


class RevenueCode(GenericModel):  # type: ignore
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
        on_delete=models.CASCADE,
        related_name="revenue_code_expense_account",
        related_query_name="revenue_code_expense_accounts",
        help_text=_("The expense account associated with the revenue code."),
        verbose_name=_("Expense Account"),
        blank=True,
        null=True,
    )
    revenue_account = models.ForeignKey(
        GeneralLedgerAccount,
        on_delete=models.CASCADE,
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
                fields=["code", "organization"],
                name="unique_revenue_code_organization",
            )
        ]

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

        super().clean()

        errors = {}

        if (
            self.expense_account
            and self.expense_account.account_type
            != GeneralLedgerAccount.AccountTypeChoices.EXPENSE
        ):
            errors["expense_account"] = _(
                "Entered account is not an expense account. Please try again."
            )

        if (
            self.revenue_account
            and self.revenue_account.account_type
            != GeneralLedgerAccount.AccountTypeChoices.REVENUE
        ):
            errors["revenue_account"] = _(
                "Entered account is not a revenue account. Please try again."
            )

        if errors:
            raise ValidationError(errors)

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

        super().save(**kwargs)

    def get_absolute_url(self) -> str:
        """RevenueCode absolute url

        Returns:
            str: RevenueCode absolute url
        """
        return reverse("accounting:revenue_code_detail", kwargs={"pk": self.pk})

    def update_revenue_code(self, **kwargs: Any):
        """Update the revenue code.

        Args:
            **kwargs: Keyword arguments

        Examples:
            >>> revenue_code = RevenueCode.objects.get(pk=uuid.uuid4())
            >>> revenue_code.update_revenue_code(code="1234", description="New Description")
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
    is_active = models.BooleanField(
        _("Active"),
        default=True,
        help_text=_("Whether the division code is active."),
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
        on_delete=models.CASCADE,
        related_name="division_code_cash_account",
        help_text=_("The cash account associated with the division code."),
        verbose_name=_("Cash Account"),
        blank=True,
        null=True,
    )
    ap_account = models.ForeignKey(
        GeneralLedgerAccount,
        on_delete=models.CASCADE,
        related_name="division_code_ap_account",
        help_text=_("The accounts payable account associated with the division code."),
        verbose_name=_("Accounts Payable Account"),
        blank=True,
        null=True,
    )
    expense_account = models.ForeignKey(
        GeneralLedgerAccount,
        on_delete=models.CASCADE,
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
        constraints = [
            models.UniqueConstraint(
                fields=["code", "organization"],
                name="unique_division_code_organization",
            )
        ]

    def __str__(self) -> str:
        """DivisionCode string representation

        Returns:
            str: DivisionCode string representation
        """
        return textwrap.wrap(self.code, 4)[0]

    def clean(self) -> None:
        """DivisionCode model validation

        Returns:
            None
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

    def get_absolute_url(self) -> str:
        """DivisionCode absolute url

        Returns:
            str: DivisionCode absolute url
        """
        return reverse("division-codes-detail", kwargs={"pk": self.pk})
