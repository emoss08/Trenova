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

from collections.abc import Generator
from typing import Any

import pytest
from django.urls import reverse
from django.utils import timezone

from billing.tests.factories import AccessorialChargeFactory
from commodities.factories import CommodityFactory
from customer.factories import CustomerFactory
from dispatch import factories
from equipment.tests.factories import EquipmentTypeFactory
from location.factories import LocationFactory
from order.tests.factories import OrderTypeFactory
from utils.models import RatingMethodChoices

pytestmark = pytest.mark.django_db


@pytest.fixture
def rate() -> Generator[Any, Any, None]:
    """
    Rate Fixture
    """
    yield factories.RateFactory()


@pytest.fixture
def rate_table() -> Generator[Any, Any, None]:
    """
    Rate Table Fixture
    """
    yield factories.RateTableFactory()


@pytest.fixture
def rate_billing_table() -> Generator[Any, Any, None]:
    """
    Rate Billing Table
    """
    yield factories.RateBillingTableFactory()


@pytest.fixture
def rate_api(api_client, organization) -> Generator[Any, Any, None]:
    """
    Rate API
    """
    customer = CustomerFactory()
    commodity = CommodityFactory()
    order_type = OrderTypeFactory()
    equipment_type = EquipmentTypeFactory()

    data = {
        "organization": organization.id,
        "customer": customer.id,
        "effective_date": timezone.now().date(),
        "expiration_date": timezone.now().date(),
        "commodity": commodity.id,
        "order_type": order_type.id,
        "equipment_type": equipment_type.id,
        "comments": "Test Rate",
    }

    yield api_client.post(
        "/api/rates/",
        data,
    )


@pytest.fixture
def rate_table_api(api_client, rate, organization) -> Generator[Any, Any, None]:
    """
    Rate Table API
    """
    origin_location = LocationFactory()
    destination_location = LocationFactory()

    data = {
        "organization": organization.id,
        "rate": rate.id,
        "description": "Test Rate Table",
        "origin_location": origin_location.id,
        "destination_location": destination_location.id,
        "rate_method": RatingMethodChoices.FLAT,
        "rate_amount": 100.00,
    }

    yield api_client.post(
        reverse("rate-tables-list"),
        data,
    )


@pytest.fixture
def rate_billing_table_api(api_client, rate, organization) -> Generator[Any, Any, None]:
    """
    Rate Billing Table API
    """
    charge_code = AccessorialChargeFactory()

    data = {
        "organization": organization.id,
        "rate": rate.id,
        "charge_code": charge_code.id,
        "description": "Test Rate Billing Table",
        "units": 1,
    }

    yield api_client.post(
        reverse("rate-billing-tables-list"),
        data,
    )
