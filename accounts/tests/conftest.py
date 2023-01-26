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

from accounts.tests.factories import JobTitleFactory, TokenFactory, UserFactory

pytestmark = pytest.mark.django_db


@pytest.fixture
def job_title():
    """
    Job title fixture
    """
    return JobTitleFactory()


@pytest.fixture
def token():
    """
    Token fixture
    """
    return TokenFactory()


@pytest.fixture
def user():
    """
    User fixture
    """
    return UserFactory()


@pytest.fixture
def user_api(api_client, organization):
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
