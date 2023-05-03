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
from django.utils import timezone
from rest_framework.test import APIClient

from location.factories import LocationFactory
from location.models import Location
from movements.models import Movement
from movements.tests.factories import MovementFactory
from order.tests.factories import OrderFactory
from organization.models import Organization
from stops import models
from stops.tests.factories import StopFactory

pytestmark = pytest.mark.django_db


@pytest.fixture
def stop() -> Generator[Any, Any, None]:
    """
    Stop Fixture
    """
    yield StopFactory()


@pytest.fixture
def movement() -> Generator[Any, Any, None]:
    """
    Movement Fixture
    """
    yield MovementFactory()


@pytest.fixture
def location() -> Generator[Any, Any, None]:
    """
    Location Fixture
    """
    yield LocationFactory()


@pytest.fixture
def order() -> Generator[Any, Any, None]:
    """
    Order fixture
    """
    yield OrderFactory()


@pytest.fixture
def stop_api(
    api_client: APIClient,
    movement: Movement,
    location: Location,
    organization: Organization,
) -> Generator[Any, Any, None]:
    """
    Stop API fixture
    """
    yield api_client.post(
        "/api/stops/",
        {
            "organization": organization.id,
            "movement": movement.id,
            "location": location.id,
            "appointment_time_window_start": timezone.now(),
            "appointment_time_window_end": timezone.now(),
            "stop_type": models.StopChoices.PICKUP,
        },
        format="json",
    )
