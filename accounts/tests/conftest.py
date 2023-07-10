# --------------------------------------------------------------------------------------------------
#  COPYRIGHT(c) 2023 MONTA                                                                         -
#                                                                                                  -
#  This file is part of Monta.                                                                     -
#                                                                                                  -
#  The Monta software is licensed under the Business Source License 1.1. You are granted the right -
#  to copy, modify, and redistribute the software, but only for non-production use or with a total -
#  of less than three server instances. Starting from the Change Date (November 16, 2026), the     -
#  software will be made available under version 2 or later of the GNU General Public License.     -
#  If you use the software in violation of this license, your rights under the license will be     -
#  terminated automatically. The software is provided "as is," and the Licensor disclaims all      -
#  warranties and conditions. If you use this license's text or the "Business Source License" name -
#  and trademark, you must comply with the Licensor's covenants, which include specifying the      -
#  Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use     -
#  Grant, and not modifying the license in any other way.                                          -
# --------------------------------------------------------------------------------------------------

from collections.abc import Generator
from typing import Any

import pytest

from accounts.tests.factories import JobTitleFactory, TokenFactory, UserFactory

pytestmark = pytest.mark.django_db


@pytest.fixture
def job_title() -> Generator[Any, Any, None]:
    """
    Job title fixture
    """
    yield JobTitleFactory()


@pytest.fixture
def token() -> Generator[Any, Any, None]:
    """
    Token fixture
    """
    yield TokenFactory()


@pytest.fixture
def user() -> Generator[Any, Any, None]:
    """
    User fixture
    """
    yield UserFactory()


@pytest.fixture
def user_api(api_client, organization) -> Generator[Any, Any, None]:
    """
    User Fixture
    """
    job_title = JobTitleFactory()

    yield api_client.post(
        "/api/users/",
        {
            "organization": f"{organization.id}",
            "username": "foobar",
            "email": "foobar@user.com",
            "profile": {
                "organization": f"{organization.id}",
                "first_name": "foo",
                "last_name": "bar",
                "address_line_1": "test address line 1",
                "city": "test",
                "state": "NC",
                "zip_code": "12345",
                "job_title": job_title.id,
            },
        },
        format="json",
    )
