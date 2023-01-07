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


class ApiTest:
    """
    A test mixin that gives some default fixtures for Monta.

    Methods:
        token: Fixture to get a token
        api_client: Fixture to get and authenticated
        client.
    """

    @pytest.fixture()
    def token(self):
        """
        Token Fixture
        """
        return TokenFactory()

    @pytest.fixture()
    def organization(self):
        """
        Organization Fixture
        """
        return OrganizationFactory()

    @pytest.fixture()
    def user(self):
        """
        User Fixture
        """
        return UserFactory()

    @pytest.fixture()
    def api_client(self, token) -> APIClient:
        """API client Fixture

        Returns:
            APIClient: Authenticated Api object
        """
        client = APIClient()
        client.credentials(HTTP_AUTHORIZATION="Token " + token.key)
        return client
