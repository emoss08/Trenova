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
from typing import Any, Generator

import pytest
from django.core.exceptions import ValidationError

from commodities import factories, models

pytestmark = pytest.mark.django_db


@pytest.fixture
def hazardous_material() -> Generator[Any, Any, None]:
    """
    Hazardous material fixture
    """
    yield factories.HazardousMaterialFactory()


def test_hazardous_material_creation(
    hazardous_material: models.HazardousMaterial,
) -> None:
    """
    Test commodity hazardous material creation
    """
    assert hazardous_material is not None


def test_hazardous_class_choices(hazardous_material: models.HazardousMaterial) -> None:
    """
    Test Unit of measure choices throws ValidationError
    when the passed choice is not valid.
    """
    with pytest.raises(ValidationError) as excinfo:
        hazardous_material.hazard_class = "invalid"
        hazardous_material.full_clean()

    assert excinfo.value.message_dict["hazard_class"] == [
        "Value 'invalid' is not a valid choice."
    ]


def test_packing_group_choices(hazardous_material: models.HazardousMaterial) -> None:
    """
    Test Packing group choice throws ValidationError
    when the passed choice is not valid.
    """
    with pytest.raises(ValidationError) as excinfo:
        hazardous_material.packing_group = "invalid"
        hazardous_material.full_clean()

    assert excinfo.value.message_dict["packing_group"] == [
        "Value 'invalid' is not a valid choice."
    ]


def test_hazardous_material_update(
    hazardous_material: models.HazardousMaterial,
) -> None:
    """
    Test commodity hazardous material update
    """
    hazardous_material.name = "New name"
    hazardous_material.save()
    assert hazardous_material.name == "New name"
