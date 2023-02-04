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

from typing import Any, Generator

import pytest
from django.utils import timezone

from location.factories import LocationFactory
from movements.tests.factories import MovementFactory
from stops.tests.factories import StopFactory
from stops import models

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
def stop_api(api_client, movement, location) -> Generator[Any, Any, None]:
    """
    Stop API fixture
    """
    yield api_client.post("/api/stops/", {
        "movement": f"{movement.id}",
        "location": f"{location.id}",
        "appointment_time": f"{timezone.now()}",
        "stop_type": models.StopChoices.PICKUP
    }, format="json")
