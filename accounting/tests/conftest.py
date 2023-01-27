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

from accounting.models import GeneralLedgerAccount
from accounting.tests.factories import GeneralLedgerAccountFactory, RevenueCodeFactory

pytestmark = pytest.mark.django_db


@pytest.fixture
def revenue_code():
    """
    Revenue Code Fixture
    """
    yield RevenueCodeFactory()


@pytest.fixture
def general_ledger_account():
    """
    Expense Account Fixture
    """
    yield GeneralLedgerAccountFactory()


@pytest.fixture
def revenue_code():
    """
    Revenue Code Fixture
    """
    yield RevenueCodeFactory()


@pytest.fixture
def expense_account():
    """
    Expense Code General Ledger Account Fixture
    """
    yield GeneralLedgerAccountFactory(
        account_type=GeneralLedgerAccount.AccountTypeChoices.EXPENSE
    )


@pytest.fixture
def revenue_account():
    """
    Revenue Code General Ledger Account Fixture
    """
    yield GeneralLedgerAccountFactory(
        account_type=GeneralLedgerAccount.AccountTypeChoices.REVENUE
    )
