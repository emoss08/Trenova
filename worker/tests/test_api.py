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

from accounts.factories import TokenFactory, UserFactory
from organization.factories import OrganizationFactory
from worker.factories import WorkerFactory

client = APIClient()


@pytest.fixture()
def user():
    """
    User Fixture
    """

    return UserFactory()


@pytest.fixture()
def token(user):
    """
    Token Fixture
    """

    return TokenFactory()


@pytest.fixture()
def organization(user):
    """
    Organization Fixture
    """

    return OrganizationFactory()


@pytest.fixture()
def worker():
    """
    Worker Fixture
    """

    return WorkerFactory()


@pytest.mark.django_db
def test_create_worker(token):
    """
    Test creating worker
    """

    client.credentials(HTTP_AUTHORIZATION="Token " + token.key)
    response = client.post(
        "/api/workers/",
        {
            "is_active": True,
            "worker_type": "EMPLOYEE",
            "first_name": "BIG",
            "last_name": "STEPPER",
            "address_line_1": "TEST",
            "city": "TEST",
            "state": "NC",
            "zip_code": "12345",
        },
        format="json",
    )
    assert response.status_code == 201


@pytest.mark.django_db
def test_create_worker_with_profile(token):
    """
    Test creating worker with profile
    """

    client.credentials(HTTP_AUTHORIZATION="Token " + token.key)
    response = client.post(
        "/api/workers/",
        {
            "is_active": True,
            "worker_type": "EMPLOYEE",
            "first_name": "BIG",
            "last_name": "STEPPER",
            "address_line_1": "TEST",
            "city": "TEST",
            "state": "NC",
            "zip_code": "12345",
            "profile": {
                "race": "TEST",
                "sex": "male",
                "date_of_birth": "2022-12-10",
                "license_number": "1234567890",
                "license_state": "NC",
                "endorsements": "N",
            },
        },
        format="json",
    )
    assert response.status_code == 201
