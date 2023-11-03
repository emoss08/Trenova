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
from rest_framework.test import APIClient

from accounts.models import Token
from accounts.tests.factories import ProfileFactory, TokenFactory, UserFactory
from organization.factories import BusinessUnitFactory, OrganizationFactory

pytestmark = pytest.mark.django_db


@pytest.fixture
def token() -> Generator[Any, Any, None]:
    """
    Token Fixture
    """
    yield TokenFactory()


@pytest.fixture
def business_unit() -> Generator[Any, Any, None]:
    """Business Unit Fixture

    Yields:
        Generator[Any, Any, None]: Business Unit object
    """
    yield BusinessUnitFactory()


@pytest.fixture
def organization() -> Generator[Any, Any, None]:
    """
    Organization Fixture
    """
    yield OrganizationFactory()


@pytest.fixture
def user() -> Generator[Any, Any, None]:
    """
    User Fixture
    """
    yield UserFactory()


@pytest.fixture
def user_profile() -> Generator[Any, Any, None]:
    """
    User Profile Fixture
    """
    yield ProfileFactory()


@pytest.fixture
def api_client(token: Token) -> APIClient:
    """API client Fixture

    Returns:
        APIClient: Authenticated Api object
    """
    client = APIClient()
    client.credentials(HTTP_AUTHORIZATION=f"Bearer {token.key}")
    yield client


@pytest.fixture
def unauthenticated_api_client() -> APIClient:
    """API client Fixture

    Returns:
        APIClient: Authenticated Api object
    """
    yield APIClient()
