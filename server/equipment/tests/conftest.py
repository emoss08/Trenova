# --------------------------------------------------------------------------------------------------
#  COPYRIGHT(c) 2024 Trenova                                                                       -
#                                                                                                  -
#  This file is part of Trenova.                                                                   -
#                                                                                                  -
#  The Trenova software is licensed under the Business Source License 1.1. You are granted the right
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
