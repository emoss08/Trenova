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

import pytest
from django.core.exceptions import ValidationError

from accounting import models

from accounting.tests.factories import RevenueCodeFactory, GeneralLedgerAccountFactory

pytestmark = pytest.mark.django_db


class TestRevenueCodeValidation:
    @pytest.fixture()
    def revenue_code(self):
        """
        Revenue Code Fixture
        """
        return RevenueCodeFactory()

    @pytest.fixture()
    def general_ledger_account(self):
        """
        Expense Account Fixture
        """
        return GeneralLedgerAccountFactory()

    def test_expense_account(self, revenue_code, general_ledger_account):
        """
        Test Whether the validation error
        is thrown if an account other than an expense account
        is passed.
        """

        general_ledger_account.account_type = (
            models.GeneralLedgerAccount.AccountTypeChoices.REVENUE
        )
        general_ledger_account.save()

        with pytest.raises(
                ValidationError, match="Entered account is not an expense account."
        ):
            revenue_code.expense_account = general_ledger_account
            revenue_code.full_clean()

    def test_revenue_account(self, revenue_code, general_ledger_account):
        """
        Test Whether the validation error
        is thrown if an account other than an expense account
        is passed.
        """

        general_ledger_account.account_type = (
            models.GeneralLedgerAccount.AccountTypeChoices.EXPENSE
        )
        general_ledger_account.save()

        with pytest.raises(
                ValidationError, match="Entered account is not a revenue account."
        ):
            revenue_code.revenue_account = general_ledger_account
            revenue_code.full_clean()
