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

from billing import models

pytestmark = pytest.mark.django_db


class TestChargeType:
    """
    Test for Charge Types
    """

    def test_list(self, charge_type):
        """
        Test Charge Type List
        """
        assert charge_type is not None

    def test_create(self, organization):
        """
        Test Create Charge Type
        """
        charge_type = models.ChargeType.objects.create(
            organization=organization,
            name="test",
            description="Test Description",
        )

        assert charge_type is not None
        assert charge_type.name == "test"
        assert charge_type.description == "Test Description"

    def test_update(self, charge_type):
        """
        Test Charge Type update
        """

        char_type = models.ChargeType.objects.get(id=charge_type.id)

        char_type.name = "maybe"
        char_type.save()

        assert char_type is not None
        assert char_type.name == "maybe"


class TestChargeTypeApi:
    """
    Test for Charge Type API
    """

    def test_get(self, api_client):
        """
        Test get Charge Type
        """
        response = api_client.get("/api/charge_types/")
        assert response.status_code == 200

    def test_get_by_id(self, api_client, organization):
        """
        Test get Charge Type by ID
        """

        _response = api_client.post(
            "/api/charge_types/",
            {
                "organization": f"{organization}",
                "name": "foob",
                "description": "Test Description",
            },
            format="json",
        )

        response = api_client.get(f"/api/charge_types/{_response.data['id']}/")

        assert response.status_code == 200
        assert response.data["name"] == "foob"
        assert response.data["description"] == "Test Description"

    def test_put(self, api_client, organization):
        """
        Test put Charge Type
        """

        _response = api_client.post(
            "/api/charge_types/",
            {
                "organization": f"{organization}",
                "name": "foob",
                "description": "Test Description",
            },
            format="json",
        )

        response = api_client.put(
            f"/api/charge_types/{_response.data['id']}/",
            {"name": "foo bar"},
            format="json",
        )

        assert response.status_code == 200
        assert response.data["name"] == "foo bar"

    def test_delete(self, api_client, organization):
        """
        Test Delete Charge Type
        """

        _response = api_client.post(
            "/api/charge_types/",
            {
                "organization": f"{organization}",
                "name": "foob",
                "description": "Test Description",
            },
            format="json",
        )

        response = api_client.delete(
            f"/api/charge_types/{_response.data['id']}/",
        )

        assert response.status_code == 204
        assert response.data is None
