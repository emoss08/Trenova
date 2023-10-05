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

from equipment.models import Tractor
from equipment.tests.factories import TractorFactory
from movements.tests.factories import MovementFactory
from organization.models import Organization
from shipment.models import Shipment
from shipment.tests.factories import ShipmentFactory
from worker.factories import WorkerFactory
from worker.models import Worker

pytestmark = pytest.mark.django_db


@pytest.fixture
def movement() -> Generator[Any, Any, None]:
    """
    Pytest Fixture for Movement
    """
    yield MovementFactory()


@pytest.fixture
def worker() -> Generator[Any, Any, None]:
    """
    Pytest Fixture for Worker
    """
    yield WorkerFactory()


@pytest.fixture
def tractor() -> Generator[Any, Any, None]:
    """
    Pytest fixture for Equipment
    """
    yield TractorFactory()


@pytest.fixture
def shipment() -> Generator[Any, Any, None]:
    """
    Pytest fixture for Order
    """
    yield ShipmentFactory()


@pytest.fixture
def movement_api(
    api_client: APIClient,
    organization: Organization,
    shipment: Shipment,
    tractor: Tractor,
    worker: Worker,
) -> Generator[Any, Any, None]:
    """
    Movement Factory
    """
    yield api_client.post(
        "/api/movements/",
        {
            "organization": f"{organization.id}",
            "shipment": f"{shipment.id}",
            "primary_worker": f"{worker.id}",
            "tractor": f"{tractor.id}",
        },
    )
