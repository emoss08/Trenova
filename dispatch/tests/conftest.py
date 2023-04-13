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
from django.urls import reverse
from django.utils import timezone
from rest_framework.test import APIClient

from billing.tests.factories import AccessorialChargeFactory
from commodities.factories import CommodityFactory
from customer.factories import CustomerFactory
from dispatch import factories
from dispatch.models import Rate
from equipment.tests.factories import EquipmentTypeFactory
from location.factories import LocationFactory
from order.tests.factories import OrderTypeFactory
from organization.models import Organization
from utils.models import RatingMethodChoices

pytestmark = pytest.mark.django_db


@pytest.fixture
def rate() -> Generator[Any, Any, None]:
    """
    Rate Fixture
    """
    yield factories.RateFactory()


@pytest.fixture
def rate_billing_table() -> Generator[Any, Any, None]:
    """
    Rate Billing Table
    """
    yield factories.RateBillingTableFactory()


@pytest.fixture
def rate_api(
    api_client: APIClient, organization: Organization
) -> Generator[Any, Any, None]:
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
def rate_table_api(
    api_client: APIClient, rate: Rate, organization: Organization
) -> Generator[Any, Any, None]:
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
