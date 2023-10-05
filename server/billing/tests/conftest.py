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

from accounting.tests.factories import RevenueCodeFactory
from billing.tests.factories import (
    AccessorialChargeFactory,
    ChargeTypeFactory,
    DocumentClassificationFactory,
)
from commodities.factories import CommodityFactory
from customer.factories import CustomerFactory
from organization.factories import EmailProfileFactory
from shipment.tests.factories import ShipmentFactory, ShipmentTypeFactory
from worker.factories import WorkerFactory

pytestmark = pytest.mark.django_db


@pytest.fixture
def document_classification() -> Generator[Any, Any, None]:
    """
    Document classification fixture
    """
    yield DocumentClassificationFactory()


@pytest.fixture
def charge_type() -> Generator[Any, Any, None]:
    """
    Charge type fixture
    """
    yield ChargeTypeFactory()


@pytest.fixture
def shipment_type() -> Generator[Any, Any, None]:
    """
    shipment type Fixture
    """
    yield ShipmentTypeFactory()


@pytest.fixture
def shipment() -> Generator[Any, Any, None]:
    """
    Order Fixture
    """
    yield ShipmentFactory()


@pytest.fixture
def revenue_code() -> Generator[Any, Any, None]:
    """
    Revenue Code Fixture
    """
    yield RevenueCodeFactory()


@pytest.fixture
def customer() -> Generator[Any, Any, None]:
    """
    Customer Fixture
    """
    yield CustomerFactory()


@pytest.fixture
def worker() -> Generator[Any, Any, None]:
    """
    Worker Fixture
    """
    yield WorkerFactory()


@pytest.fixture
def commodity() -> Generator[Any, Any, None]:
    """
    Commodity Fixture
    """
    yield CommodityFactory()


@pytest.fixture
def email_profile() -> Generator[Any, Any, None]:
    """
    Email Profile fixture
    """
    yield EmailProfileFactory()


@pytest.fixture
def accessorial_charge() -> Generator[Any, Any, None]:
    """
    Accessorial charge fixture
    """
    yield AccessorialChargeFactory()


@pytest.fixture
def charge_type_api(api_client, organization) -> Generator[Any, Any, None]:
    """
    Charge type fixture
    """
    yield api_client.post(
        "/api/charge_types/",
        {
            "organization": organization.id,
            "name": "foob",
            "description": "Test Description",
        },
        format="json",
    )
