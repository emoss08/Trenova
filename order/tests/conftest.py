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

from accounting.tests.factories import RevenueCodeFactory
from accounts.tests.factories import TokenFactory, UserFactory
from billing.tests.factories import (
    AccessorialChargeFactory,
    DocumentClassificationFactory,
)
from customer.factories import CustomerFactory
from dispatch.factories import CommentTypeFactory
from equipment.tests.factories import EquipmentTypeFactory
from order.tests.factories import (
    AdditionalChargeFactory,
    OrderCommentFactory,
    OrderDocumentationFactory,
    OrderFactory,
    OrderTypeFactory,
    ReasonCodeFactory,
)
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
def api_client(token):
    """API client Fixture

    Returns:
        APIClient: Authenticated Api object
    """
    client = APIClient()
    client.credentials(HTTP_AUTHORIZATION=f"Token {token.key}")
    return client


@pytest.fixture
def order_type():
    """
    Pytest Fixture for order Type
    """
    yield OrderTypeFactory()


@pytest.fixture
def order():
    """
    Pytest Fixture for Order
    """
    yield OrderFactory()


@pytest.fixture
def document_classification():
    """
    Pytest Fixture for Document Classification
    """
    yield DocumentClassificationFactory()


@pytest.fixture
def reason_code():
    """
    Pytest Fixture for Reason Code
    """
    yield ReasonCodeFactory()


@pytest.fixture
def order_document():
    """
    Pytest Fixture for Order Documentation
    """
    yield OrderDocumentationFactory()


@pytest.fixture
def additional_charge():
    """
    Pytest Fixture for order Type
    """
    yield AdditionalChargeFactory()


@pytest.fixture
def accessorial_charge():
    """
    Pytest Fixture for Accessorial Charge
    """
    yield AccessorialChargeFactory()


@pytest.fixture
def revenue_code():
    """
    Pytest Fixture for Revenue Code
    """
    yield RevenueCodeFactory()


@pytest.fixture()
def customer():
    """
    Pytest Fixture for Customer
    """
    yield CustomerFactory()


@pytest.fixture
def equipment_type():
    """
    Pytest Fixture for Equipment Type
    """
    yield EquipmentTypeFactory()


@pytest.fixture
def order_comment():
    """
    Pytest Fixture for Order Comment
    """
    yield OrderCommentFactory()


@pytest.fixture
def comment_type():
    """
    Pytest Fixture for Comment Type
    """
    yield CommentTypeFactory()
