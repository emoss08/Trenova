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
from django.urls import reverse

from accounting import models

pytestmark = pytest.mark.django_db


class TestGeneralLedgerAccount:
    """
    Class to test General Ledger Account
    """

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
            account_classification=models.GeneralLedgerAccount.AccountClassificationChoices.ACCOUNTS_PAYABLE,
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


class TestGeneralLedgerAccountApi:
    """
    Class for the General Ledger account api
    """

    def test_get(self, api_client):
        """
        Test get General Ledger accounts
        """
        response = api_client.get(reverse("general_ledger_accounts-list"))
        assert response.status_code == 200

    def test_get_by_id(self, api_client, gl_account_api):
        """
        Test get General Ledger account by ID
        """
        response = api_client.get(
            reverse(
                "general_ledger_accounts-detail",
                kwargs={"pk": gl_account_api.data["id"]},
            )
        )
        assert response.status_code == 200
        assert response.data["account_number"] == gl_account_api.data["account_number"]
        assert response.data["account_type"] == gl_account_api.data["account_type"]
        assert response.data["description"] == gl_account_api.data["description"]

    def test_put(self, api_client, gl_account_api):
        """
        Test put General Ledger Account
        """
        response = api_client.put(
            reverse(
                "general_ledger_accounts-detail",
                kwargs={"pk": gl_account_api.data["id"]},
            ),
            {
                "account_number": "2345-2345-2345-2345",
                "description": "Another Test Description",
                "account_type": f"{models.GeneralLedgerAccount.AccountTypeChoices.EXPENSE}",
                "cash_flow_type": f"{models.GeneralLedgerAccount.CashFlowTypeChoices.FINANCING}",
                "account_sub_type": f"{models.GeneralLedgerAccount.AccountSubTypeChoices.CURRENT_ASSET}",
                "account_classification": f"{models.GeneralLedgerAccount.AccountClassificationChoices.ACCOUNTS_RECEIVABLE}",
            },
        )
        assert response.status_code == 200
        assert response.data["account_number"] == "2345-2345-2345-2345"
        assert (
            response.data["account_type"]
            == models.GeneralLedgerAccount.AccountTypeChoices.EXPENSE
        )
        assert response.data["description"] == "Another Test Description"
        assert (
            response.data["cash_flow_type"]
            == models.GeneralLedgerAccount.CashFlowTypeChoices.FINANCING
        )
        assert (
            response.data["account_sub_type"]
            == models.GeneralLedgerAccount.AccountSubTypeChoices.CURRENT_ASSET
        )
        assert (
            response.data["account_classification"]
            == models.GeneralLedgerAccount.AccountClassificationChoices.ACCOUNTS_RECEIVABLE
        )

    def test_delete(self, api_client, gl_account_api):
        """
        Test delete general Ledger account
        """
        response = api_client.delete(
            reverse(
                "general_ledger_accounts-detail",
                kwargs={"pk": gl_account_api.data["id"]},
            )
        )

        assert response.status_code == 200
        assert not response.data


class TestGeneralLedgerAccountValidation:
    """
    Class for the General Ledger Account Validation.
    """

    def test_account_number(self, general_ledger_account):
        """
        Test Whether the validation error is thrown
        if the entered account_number value is not a
        regex match.
        """

        with pytest.raises(ValidationError) as excinfo:
            general_ledger_account.account_number = "00000-2323411-124141"
            general_ledger_account.full_clean()

        assert excinfo.value.message_dict["account_number"] == [
            "Account number must be in the format 0000-0000-0000-0000."
        ]

    def test_unique_account_numer(self, general_ledger_account):
        """
        Test creating a General Ledger account with the same account number
        throws ValidationError.
        """
        models.GeneralLedgerAccount.objects.create(
            organization=general_ledger_account.organization,
            account_number="1234-1234-1234-1234",
            account_type=models.GeneralLedgerAccount.AccountTypeChoices.ASSET,
            description="Another Description",
            cash_flow_type=models.GeneralLedgerAccount.CashFlowTypeChoices.FINANCING,
            account_sub_type=models.GeneralLedgerAccount.AccountSubTypeChoices.CURRENT_ASSET,
            account_classification=models.GeneralLedgerAccount.AccountClassificationChoices.ACCOUNTS_PAYABLE,
        )
        with pytest.raises(ValidationError) as excinfo:
            models.GeneralLedgerAccount.objects.create(
                organization=general_ledger_account.organization,
                account_number="1234-1234-1234-1234",
                account_type=models.GeneralLedgerAccount.AccountTypeChoices.ASSET,
                description="Another Description",
                cash_flow_type=models.GeneralLedgerAccount.CashFlowTypeChoices.FINANCING,
                account_sub_type=models.GeneralLedgerAccount.AccountSubTypeChoices.CURRENT_ASSET,
                account_classification=models.GeneralLedgerAccount.AccountClassificationChoices.ACCOUNTS_PAYABLE,
            )

        assert excinfo.value.message_dict["account_number"] == [
            "An account with this account number already exists. Please try again."
        ]
