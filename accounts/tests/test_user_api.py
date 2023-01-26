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

pytestmark = pytest.mark.django_db


class TestUserAPI:
    def test_get(self, token, api_client):
        """
        Test get users
        """
        response = api_client.get("/api/users/")
        assert response.status_code == 200

    def test_get_by_id(self, api_client, user_api, token):
        """
        Test get user by ID
        """
        response = api_client.get(f"/api/users/{user_api.data['id']}/")
        assert response.status_code == 200

    def test_put(self, token, user_api, api_client):
        """
        Test Put request
        """
        response = api_client.put(
            f"/api/users/{user_api.data['id']}/",
            {
                "username": "test",
                "email": "test@test.com",
                "profile": {
                    "first_name": "test",
                    "last_name": "user",
                    "address_line_1": "test",
                    "city": "test",
                    "state": "NC",
                    "zip_code": "12345",
                },
            },
            format="json",
        )

        assert response.status_code == 200
        assert response.data["username"] == "test"
        assert response.data["email"] == "test@test.com"
        assert response.data["profile"]["first_name"] == "test"
        assert response.data["profile"]["last_name"] == "user"
        assert response.data["profile"]["address_line_1"] == "test"
        assert response.data["profile"]["city"] == "test"
        assert response.data["profile"]["state"] == "NC"
        assert response.data["profile"]["zip_code"] == "12345"

    def test_delete(self, user_api, token, api_client):
        """
        Test delete user
        """
        response = api_client.delete(f"/api/users/{user_api.data['id']}/")
        assert response.status_code == 204
        assert response.data is None
