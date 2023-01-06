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

from accounting.tests.factories import GeneralLedgerAccountFactory

pytestmark = pytest.mark.django_db


class TestGeneralLedgerAccountValidation:
    @pytest.fixture()
    def general_ledger_account(self):
        """
        General Ledger Account Fixture
        """
        return GeneralLedgerAccountFactory()

    def test_account_number(self, general_ledger_account):
        """
        Test Whether the validation error is thrown
        if the entered account_number value is not a
        regex match.
        """

        with pytest.raises(
            ValidationError,
            match="Account number must be in the format 0000-0000-0000-0000.",
        ):
            general_ledger_account.account_number = "00000-2323411-1241412312312"
            general_ledger_account.full_clean()
