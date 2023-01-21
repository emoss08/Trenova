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
from billing.tests.factories import DocumentClassificationFactory, ChargeTypeFactory
from organization.factories import OrganizationFactory


pytestmark = pytest.mark.django_db


@pytest.fixture
def token():
    """
    Token Fixture
    """
    token = TokenFactory()
    yield token


@pytest.fixture
def organization():
    """
    Organization Fixture
    """
    organization = OrganizationFactory()
    yield organization


@pytest.fixture
def user():
    """
    User Fixture
    """
    user = UserFactory()
    yield user


@pytest.fixture
def api_client(token):
    """API client Fixture

    Returns:
        APIClient: Authenticated Api object
    """
    client = APIClient()
    client.credentials(HTTP_AUTHORIZATION="Token " + token.key)
    yield client


@pytest.fixture
def document_classification():
    """
    Document classification fixture
    """
    document_classification = DocumentClassificationFactory()
    yield document_classification


@pytest.fixture()
def charge_type():
    """
    Charge type fixture
    """
    charge_type = ChargeTypeFactory()
    yield charge_type
