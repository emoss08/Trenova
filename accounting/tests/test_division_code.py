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

pytestmark = pytest.mark.django_db


class TestDivisionCode:
    """
    Test for Division code
    """

    def test_list(self, division_code):
        """
        Test Division Code List
        """
        assert division_code is not None

    def test_create(self, organization, ap_account, expense_account, cash_account):
        """
        Test Division Code Creation
        """

        div_code = models.DivisionCode.objects.create(
            organization=organization,
            is_active=True,
            code="NEW",
            description="Test Description",
            cash_account=cash_account,
            ap_account=ap_account,
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


class TestDivisionCodeApi:
    """
    Test for Division Code API
    """

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

        cash_account_data = api_client.post(
            "/api/gl_accounts/",
            {
                "is_active": True,
                "account_number": "7000-0000-0000-0000",
                "description": "Foo bar",
                "account_type": "ASSET",
                "account_classification": "CASH",
            },
            format="json",
        )

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
            {
                "code": "foob",
                "is_active": False,
                "description": "Another Description",
                "cash_account": f"{cash_account_data.data['id']}",
            },
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
        assert response.status_code == 200
        assert response.data is None
