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

from accounts.tests.factories import JobTitleFactory, TokenFactory
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
def api_client(token):
    """API client Fixture

    Returns:
        APIClient: Authenticated Api object
    """
    client = APIClient()
    client.credentials(HTTP_AUTHORIZATION=f"Token {token.key}")
    yield client


@pytest.fixture
def job_title():
    """
    Job title fixture
    """
    yield JobTitleFactory()


@pytest.fixture
def user(api_client, organization):
    """
    User Fixture
    """
    yield api_client.post(
        "/api/users/",
        {
            "organization": f"{organization}",
            "username": "foobar",
            "email": "foobar@user.com",
            "password": "trashuser12345%",
            "profile": {
                "organization": f"{organization}",
                "first_name": "foo",
                "last_name": "bar",
                "address_line_1": "test address line 1",
                "city": "test",
                "state": "NC",
                "zip_code": "12345",
            },
        },
        format="json",
    )
