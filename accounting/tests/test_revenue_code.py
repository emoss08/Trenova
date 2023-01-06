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

from accounting.tests.factories import RevenueCodeFactory, GeneralLedgerAccountFactory
from accounting import models

pytestmark = pytest.mark.django_db


class TestRevenueCode:
    @pytest.fixture()
    def revenue_code(self):
        """
        Revenue Code Fixture
        """
        return RevenueCodeFactory()

    @pytest.fixture()
    def expense_account(self):
        """
        Expense Code General Ledger Account Fixture
        """
        return GeneralLedgerAccountFactory()

    @pytest.fixture()
    def revenue_account(self):
        """
        Revenue Code General Ledger Account Fixture
        """
        return GeneralLedgerAccountFactory()

    def test_list(self, revenue_code):
        """
        Test Revenue code list
        """
        assert revenue_code is not None

    def test_create(self, revenue_code, expense_account, revenue_account):
        """
        Test Revenue code creation
        """

        expense_account.account_type = (
            models.GeneralLedgerAccount.AccountTypeChoices.EXPENSE
        )
        expense_account.save()

        revenue_account.account_type = (
            models.GeneralLedgerAccount.AccountTypeChoices.REVENUE
        )
        revenue_account.save()

        rev_code = models.RevenueCode.objects.create(
            organization=expense_account.organization,
            code="TEST",
            description="Another Description",
            expense_account=expense_account,
            revenue_account=revenue_account,
        )

        assert rev_code is not None
