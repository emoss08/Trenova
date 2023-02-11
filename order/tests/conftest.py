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
from django.utils import timezone

from accounting.tests.factories import RevenueCodeFactory
from billing.tests.factories import (
    AccessorialChargeFactory,
    DocumentClassificationFactory,
)
from customer.factories import CustomerFactory
from dispatch.factories import CommentTypeFactory
from equipment.tests.factories import EquipmentTypeFactory
from location.factories import LocationFactory
from order.tests.factories import (
    AdditionalChargeFactory,
    OrderCommentFactory,
    OrderDocumentationFactory,
    OrderFactory,
    OrderTypeFactory,
    ReasonCodeFactory,
)

pytestmark = pytest.mark.django_db


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


@pytest.fixture
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


@pytest.fixture
def origin_location():
    """
    Pytest Fixture for Origin Location
    """
    return LocationFactory()


@pytest.fixture
def destination_location():
    """
    Pytest Fixture for Destination Location
    """
    return LocationFactory()


@pytest.fixture
def order_api(
    api_client,
    organization,
    order_type,
    revenue_code,
    origin_location,
    destination_location,
    customer,
    equipment_type,
    user,
):
    """
    Pytest Fixture for Reason Code
    """
    return api_client.post(
        "/api/orders/",
        {
            "organization": f"{organization.id}",
            "order_type": f"{order_type.id}",
            "revenue_code": f"{revenue_code.id}",
            "origin_location": f"{origin_location.id}",
            "origin_appointment": f"{timezone.now()}",
            "destination_location": f"{destination_location.id}",
            "destination_appointment": f"{timezone.now()}",
            "freight_charge_amount": 100.00,
            "customer": f"{customer.id}",
            "equipment_type": f"{equipment_type.id}",
            "entered_by": f"{user.id}",
            "bol_number": "newbol",
        },
        format="json",
    )


@pytest.fixture
def additional_charge_api(api_client, user, organization, order, accessorial_charge):
    """
    Additional Charge Factory
    """
    yield api_client.post(
        "/api/additional_charges/",
        {
            "organization": f"{organization.id}",
            "order": f"{order.id}",
            "charge": f"{accessorial_charge.id}",
            "charge_amount": 123.00,
            "unit": 2,
            "entered_by": f"{user.id}",
        },
        format="json",
    )


@pytest.fixture
def order_comment_api(order_api, user, comment_type, api_client):
    """
    Pytest Fixture for Order Comment
    """
    return api_client.post(
        "/api/order_comments/",
        {
            "order": f"{order_api.data['id']}",
            "comment_type": f"{comment_type.id}",
            "comment": "IM HAPPY YOU'RE HERE",
            "entered_by": f"{user.id}",
        },
        format="json",
    )


@pytest.fixture
def order_documentation_api(api_client, order, document_classification, organization):
    """
    Pytest Fixture for Order Documentation
    """

    with open("order/tests/files/dummy.pdf", "rb") as test_file:
        yield api_client.post(
            "/api/order_documents/",
            {
                "organization": f"{organization}",
                "order": f"{order.id}",
                "document": test_file,
                "document_class": f"{document_classification.id}",
            },
        )


@pytest.fixture
def order_type_api(api_client):
    """
    Order Type Factory
    """
    return api_client.post(
        "/api/order_types/",
        {"name": "Foo Bar", "description": "Foo Bar", "is_active": True},
    )


@pytest.fixture
def reason_code_api(api_client):
    """
    Reason Code Factory
    """
    return api_client.post(
        "/api/reason_codes/",
        {
            "code": "NEWT",
            "description": "Foo Bar",
            "is_active": True,
            "code_type": "VOIDED",
        },
    )
