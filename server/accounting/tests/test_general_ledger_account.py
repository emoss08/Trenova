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

from accounting.factories import GeneralLedgerAccountFactory, RevenueCodeFactory


@pytest.fixture()
def general_ledger_account():
    """
    General Ledger Account fixture
    """
    return GeneralLedgerAccountFactory()


@pytest.fixture()
def revenue_code():
    """
    Revenue Code fixture
    """
    return RevenueCodeFactory()


@pytest.mark.django_db
def test_general_ledger_account_creation(general_ledger_account):
    """
    Test general ledger account creation
    """
    assert general_ledger_account is not None


@pytest.mark.django_db
def test_general_ledger_account_update(general_ledger_account):
    """
    Test general ledger account update
    """
    general_ledger_account.account_number = "1234-1234-1234-1234"
    general_ledger_account.save()
    assert general_ledger_account.account_number == "1234-1234-1234-1234"
