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
from django.core.exceptions import ValidationError

from commodities.factories import HazardousMaterialFactory


@pytest.fixture()
def hazardous_material():
    """
    Hazardous material fixture
    """
    return HazardousMaterialFactory()


@pytest.mark.django_db
def test_hazardous_material_creation(hazardous_material):
    """
    Test commodity hazardous material creation
    """
    assert hazardous_material is not None


@pytest.mark.django_db
def test_hazardous_class_choices(hazardous_material):
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



@pytest.mark.django_db
def test_packing_group_choices(hazardous_material):
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

@pytest.mark.django_db
def test_hazardous_material_update(hazardous_material):
    """
    Test commodity hazardous material update
    """
    hazardous_material.name = "New name"
    hazardous_material.save()
    assert hazardous_material.name == "New name"
