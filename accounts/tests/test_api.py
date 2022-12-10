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

import json

import pytest
from rest_framework.test import APIClient

from accounts.factories import TokenFactory, UserFactory
from organization.factories import OrganizationFactory

client = APIClient()


@pytest.fixture()
def user():
    """
    User fixture
    """
    return UserFactory()


@pytest.fixture()
def token(user):
    """
    Token fixture
    """
    return TokenFactory(user=user)


@pytest.fixture()
def organization(user):
    """
    Organization fixture
    """
    return OrganizationFactory()


@pytest.mark.django_db
def test_get_user(token):
    """
    Test get user
    """
    client.credentials(HTTP_AUTHORIZATION="Token " + token.key)
    response = client.get("/api/users/")
    assert response.status_code == 200


@pytest.mark.django_db
def test_create_user(token, organization):
    """
    Test create user
    """
    client.credentials(HTTP_AUTHORIZATION="Token " + token.key)
    response = client.post(
        "/api/users/",
        {
            "organization": organization.id,
            "username": "test",
            "password": "test12345",
            "email": "test@test.com",
        },
    )
    print(response.data)

    assert response.status_code == 201


@pytest.mark.django_db
def test_update_user(user, token):
    """
    Test update user
    """
    client.credentials(HTTP_AUTHORIZATION="Token " + token.key)
    response = client.put(
        f"/api/users/{str(user.id)}/",
        {
            "organization": user.organization.id,
            "username": "test33",
            "password": user.password,
            "email": user.email,
        },
    )
    assert json.loads(response.content)["username"] == "test33"
    assert response.status_code == 200


@pytest.mark.django_db
def test_patch_user(user, token):
    """
    Test patch user
    """
    client.credentials(HTTP_AUTHORIZATION="Token " + token.key)
    response = client.patch(
        f"/api/users/{str(user.id)}/",
        {
            "organization": user.organization.id,
            "username": "test33",
            "password": user.password,
            "email": user.email,
        },
    )
    assert json.loads(response.content)["username"] == "test33"
    assert response.status_code == 200


@pytest.mark.django_db
def test_delete_user(user, token):
    """
    Test delete user
    """
    client.credentials(HTTP_AUTHORIZATION="Token " + token.key)
    response = client.delete(f"/api/users/{str(user.id)}/")
    assert response.status_code == 204
