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

from commodities.factories import CommodityFactory


@pytest.fixture
def commodity():
    """
    Commodity fixture
    """
    yield CommodityFactory()


@pytest.mark.django_db
def test_commodity_creation(commodity):
    """
    Test commodity creation
    """
    assert commodity is not None


@pytest.mark.django_db
def test_unit_of_measure_choices(commodity):
    """
    Test Unit of measure choices throws ValidationError
    when the passed choice is not valid.
    """
    with pytest.raises(ValidationError) as excinfo:
        commodity.unit_of_measure = "invalid"
        commodity.full_clean()

    assert excinfo.value.message_dict["unit_of_measure"] == [
        "Value 'invalid' is not a valid choice."
    ]


@pytest.mark.django_db
def test_commodity_update(commodity):
    """
    Test commodity update
    """
    commodity.name = "New name"
    commodity.save()
    assert commodity.name == "New name"


@pytest.mark.django_db
def test_commodity_is_hazmat_if_hazmat_class(commodity):
    """
    Test commodity hazardous material creation
    """
    assert commodity.is_hazmat is True
