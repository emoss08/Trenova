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

from accounting.tests.factories import GeneralLedgerAccountFactory
from accounting import models

pytestmark = pytest.mark.django_db


class TestGeneralLedgerAccount:
    @pytest.fixture()
    def general_ledger_account(self):
        """
        General Ledger Account fixture
        """
        return GeneralLedgerAccountFactory()

    def test_list(self, general_ledger_account):
        """
        Test general ledger account list
        """
        assert general_ledger_account is not None

    def test_create(self, general_ledger_account):
        """
        Test general ledger account creation
        """
        gl_account = models.GeneralLedgerAccount.objects.create(
            organization=general_ledger_account.organization,
            account_number="1234-1234-1234-1234",
            account_type=models.GeneralLedgerAccount.AccountTypeChoices.ASSET,
            description="Another Description",
            cash_flow_type=models.GeneralLedgerAccount.CashFlowTypeChoices.FINANCING,
            account_sub_type=models.GeneralLedgerAccount.AccountSubTypeChoices.CURRENT_ASSET,
            account_classification=models.GeneralLedgerAccount.AccountClassificationChoices.ACCOUNTS_PAYABLE
        )

        assert gl_account is not None
        assert gl_account.account_number == "1234-1234-1234-1234"

    def test_update(self, general_ledger_account):
        """
        Test general ledger account update
        """
        general_ledger_account.account_number = "1234-1234-1234-1234"
        general_ledger_account.save()
        assert general_ledger_account.account_number == "1234-1234-1234-1234"
