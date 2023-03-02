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
from django.urls import reverse
from rest_framework.test import APIClient

from accounts.models import User
from accounts.tests.factories import TokenFactory, UserFactory
from organization.factories import OrganizationFactory

pytestmark = pytest.mark.django_db


@pytest.fixture
def token():
    """
    Token Fixture
    """
    yield TokenFactory()


@pytest.fixture
def organization():
    """
    Organization Fixture
    """
    yield OrganizationFactory()


@pytest.fixture
def user():
    """
    User Fixture
    """
    yield UserFactory()


@pytest.fixture
def api_client(token, organization):
    """API client Fixture

    Returns:
        APIClient: Authenticated Api object
    """
    client = APIClient()

    user = User.objects.create_user(
        organization=organization,
        username="test",
        password="password",
        email="testuser@testing.com",
    )

    client.post(
        reverse("knox_login"), data={"username": user.username, "password": "password"}
    )
    client.credentials(HTTP_AUTHORIZATION=f"Token {token.token_key}")
    yield client
