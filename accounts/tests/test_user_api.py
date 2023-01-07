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
from rest_framework.test import APIClient

from accounts.tests.factories import TokenFactory, UserFactory
from organization.factories import OrganizationFactory

client = APIClient()

pytestmark = pytest.mark.django_db


class TestUserAPI:
    @pytest.fixture()
    def user(self):
        """
        User fixture
        """
        return UserFactory()

    @pytest.fixture()
    def token(self, user):
        """
        Token fixture
        """
        return TokenFactory(user=user)

    @pytest.fixture()
    def organization(self, user):
        """
        Organization fixture
        """
        return OrganizationFactory()

    def test_get(self, token):
        """
        Test get users
        """
        client.credentials(HTTP_AUTHORIZATION="Token " + token.key)
        response = client.get("/api/users/")
        assert response.status_code == 200

    def test_get_by_id(self, user, token):
        """
        Test get user by ID
        """
        client.credentials(HTTP_AUTHORIZATION="Token " + token.key)
        response = client.get(f"/api/users/{user.id}/")
        assert response.status_code == 200

    def test_post(self, token, organization):
        """
        Test create user
        """
        client.credentials(HTTP_AUTHORIZATION="Token " + token.key)
        response = client.post(
            "/api/users/",
            {
                "organization": organization.id,
                "username": "test",
                "email": "test@test.com",
            },
            format="json",
        )
        assert response.status_code == 201
        assert response.data["username"] == "test"
        assert response.data["email"] == "test@test.com"

    def test_put(self, token, user):
        """
        Test Put request
        """
        client.credentials(HTTP_AUTHORIZATION="Token " + token.key)
        response = client.put(
            f"/api/users/{user.id}/",
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

    def test_delete(self, user, token):
        """
        Test delete user
        """
        client.credentials(HTTP_AUTHORIZATION="Token " + token.key)
        response = client.delete(f"/api/users/{user.id}/")
        assert response.status_code == 204
        assert response.data is None
