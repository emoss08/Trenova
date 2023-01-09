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

from accounting import models
from accounting.tests.factories import DivisionCodeFactory, GeneralLedgerAccountFactory
from utils.tests import ApiTest, UnitTest


class TestDivisionCode(UnitTest):
    """
    Test for Division code
    """

    @pytest.fixture()
    def division_code(self):
        """
        Division Code Factory
        """
        return DivisionCodeFactory()

    @pytest.fixture()
    def general_ledger_account(self):
        """
        General Ledger Account Factory
        """
        return GeneralLedgerAccountFactory()

    @pytest.fixture()
    def expense_account(self):
        """
        Expense Account from GL Account Factory
        """
        return GeneralLedgerAccountFactory(
            account_type=models.GeneralLedgerAccount.AccountTypeChoices.EXPENSE
        )

    def test_list(self, division_code):
        """
        Test Division Code List
        """
        assert division_code is not None

    def test_create(self, organization, general_ledger_account, expense_account):
        """
        Test Division Code Creation
        """

        div_code = models.DivisionCode.objects.create(
            organization=organization,
            is_active=True,
            code="NEW",
            description="Test Description",
            cash_account=general_ledger_account,
            ap_account=general_ledger_account,
            expense_account=expense_account,
        )

        assert div_code is not None
        assert div_code.is_active is True
        assert div_code.code == "NEW"
        assert div_code.description == "Test Description"

    def test_update(self, division_code):
        """
        Test Division Code update
        """

        div_code = models.DivisionCode.objects.filter(id=division_code.id).get()

        div_code.code = "FOOB"

        div_code.save()

        assert div_code is not None
        assert div_code.code == "FOOB"


class TestDivisionCodeApi(ApiTest):
    """
    Test for Division Code API
    """

    @pytest.fixture()
    def division_code(self):
        """
        Division Code Factory
        """
        return DivisionCodeFactory()

    def test_get(self, api_client):
        """
        Test get Division Code
        """
        response = api_client.get("/api/division_codes/")
        assert response.status_code == 200

    def test_get_by_id(self, api_client, organization):
        """
        Test get Division Code by ID
        """
        _response = api_client.post(
            "/api/division_codes/",
            {
                "organization": f"{organization}",
                "is_active": True,
                "code": "Test",
                "description": "Test Description",
            },
            format="json",
        )

        response = api_client.get(f"/api/division_codes/{_response.data['id']}/")

        assert response.status_code == 200
        assert response.data["is_active"] is True
        assert response.data["code"] == "Test"
        assert response.data["description"] == "Test Description"

    def test_put(self, api_client, organization):
        """
        Test put Division Code
        """

        _response = api_client.post(
            "/api/division_codes/",
            {
                "organization": f"{organization}",
                "is_active": True,
                "code": "Test",
                "description": "Test Description",
            },
            format="json",
        )

        response = api_client.put(
            f"/api/division_codes/{_response.data['id']}/",
            {"code": "foob", "is_active": False, "description": "Another Description"},
            format="json",
        )

        assert response.status_code == 200
        assert response.data["code"] == "foob"
        assert response.data["is_active"] is False
        assert response.data["description"] == "Another Description"

    def test_delete(self, api_client, organization):
        """
        Test Delete Division Code
        """

        _response = api_client.post(
            "/api/division_codes/",
            {
                "organization": f"{organization}",
                "is_active": True,
                "code": "Test",
                "description": "Test Description",
            },
            format="json",
        )

        response = api_client.delete(f"/api/division_codes/{_response.data['id']}/")

        assert response.status_code == 204
        assert response.data is None
