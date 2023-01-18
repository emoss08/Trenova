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

from order import models

pytestmark = pytest.mark.django_db

class TestReasonCode:
    """
    Class to test Reason Code
    """

    def test_list(self, reason_code):
        """
        Test Reason Code list
        """
        assert reason_code is not None

    def test_create(self, organization):
        """
        Test Reason Code Create
        """
        r_code = models.ReasonCode.objects.create(
            organization=organization,
            is_active=True,
            code="foobo",
            description="foo bar",
            code_type="VOIDED",
        )

        assert r_code is not None
        assert r_code.is_active is True
        assert r_code.code == "foobo"
        assert r_code.description == "foo bar"
        assert r_code.code_type == "VOIDED"

    def test_update(self, reason_code):
        """
        Test order type update
        """

        r_code = models.ReasonCode.objects.get(id=reason_code.id)

        r_code.code = "NEWTY"

        r_code.save()

        assert r_code is not None
        assert r_code.code == "NEWTY"


class TestReasonCodeAPI:
    """
    Test for Reason Code API
    """

    @pytest.fixture
    def reason_code(self, api_client):
        """
        Reason Code Factory
        """
        return api_client.post(
            "/api/reason_codes/",
            {
                "code": "NEWT",
                "description": "Foo Bar",
                "is_active": True,
                "code_type": "VOIDED",
            },
        )

    def test_get(self, api_client):
        """
        Test get Reason Code
        """
        response = api_client.get("/api/reason_codes/")
        assert response.status_code == 200

    def test_get_by_id(self, api_client, reason_code):
        """
        Test get Reason Code by id
        """
        response = api_client.get(f"/api/reason_codes/{reason_code.data['id']}/")

        assert response.status_code == 200
        assert response.data["code"] == "NEWT"
        assert response.data["description"] == "Foo Bar"
        assert response.data["is_active"] is True
        assert response.data["code_type"] == "VOIDED"

    def test_put(self, api_client, reason_code):
        """
        Test put Reason Code
        """
        response = api_client.put(
            f"/api/reason_codes/{reason_code.data['id']}/",
            {
                "code": "FOBO",
                "description": "New Description",
                "is_active": False,
                "code_type": "VOIDED",
            },
        )

        assert response.status_code == 200
        assert response.data["code"] == "FOBO"
        assert response.data["description"] == "New Description"
        assert response.data["is_active"] is False

    def test_delete(self, api_client, reason_code):
        """
        Test Delete Reason Code
        """
        response = api_client.delete(f"/api/reason_codes/{reason_code.data['id']}/")

        assert response.status_code == 204
        assert response.data is None
