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
    DocumentClassificationFactory,
    AccessorialChargeFactory,
)
from customer.factories import CustomerFactory
from dispatch.factories import CommentTypeFactory
from equipment.tests.factories import EquipmentTypeFactory
from order.tests.factories import (
    OrderTypeFactory,
    ReasonCodeFactory,
    OrderFactory,
    OrderDocumentationFactory,
    AdditionalChargeFactory,
    OrderCommentFactory,
)
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
    return client


@pytest.fixture
def order_type():
    """
    Pytest Fixture for order Type
    """
    order_type = OrderTypeFactory()
    yield order_type


@pytest.fixture
def order():
    """
    Pytest Fixture for Order
    """
    order = OrderFactory()
    yield order


@pytest.fixture
def document_classification():
    """
    Pytest Fixture for Document Classification
    """
    document_classification = DocumentClassificationFactory()
    yield document_classification


@pytest.fixture
def reason_code():
    """
    Pytest Fixture for Reason Code
    """
    reason_code = ReasonCodeFactory()
    yield reason_code


@pytest.fixture
def order_document():
    """
    Pytest Fixture for Order Documentation
    """
    order_document = OrderDocumentationFactory()
    yield order_document


@pytest.fixture
def additional_charge():
    """
    Pytest Fixture for order Type
    """
    additional_charge = AdditionalChargeFactory()
    yield additional_charge


@pytest.fixture
def accessorial_charge():
    """
    Pytest Fixture for Accessorial Charge
    """
    accessorial_charge = AccessorialChargeFactory()
    yield accessorial_charge


@pytest.fixture
def revenue_code():
    """
    Pytest Fixture for Revenue Code
    """
    revenue_code = RevenueCodeFactory()
    yield revenue_code


@pytest.fixture()
def customer():
    """
    Pytest Fixture for Customer
    """
    customer = CustomerFactory()
    yield customer


@pytest.fixture
def equipment_type():
    """
    Pytest Fixture for Equipment Type
    """
    equipment_type = EquipmentTypeFactory()
    yield equipment_type


@pytest.fixture
def order_comment():
    """
    Pytest Fixture for Order Comment
    """
    order_comment = OrderCommentFactory()
    yield order_comment


@pytest.fixture
def comment_type():
    """
    Pytest Fixture for Comment Type
    """
    comment_type = CommentTypeFactory()
    yield comment_type
