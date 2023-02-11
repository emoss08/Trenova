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

from equipment.tests.factories import EquipmentTypeFactory

pytestmark = pytest.mark.django_db

@pytest.fixture
def equipment_type() -> Generator[Any, Any, None]:
    """
    EquipmentType Fixture
    """

    yield EquipmentTypeFactory()


@pytest.fixture
def equipment_type_api(api_client) -> Generator[Any, Any, None]:
    """
    Equipment Type API Fixture
    """
    post_data = {
        "name": "test_equipment_type",
        "description": "Test Equipment Type Description",
    }
    yield api_client.post(
        reverse("equipment-types-list"),
        post_data,
        format="json",
    )