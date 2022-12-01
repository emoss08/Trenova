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

from organization.factories import OrganizationFactory


@pytest.fixture()
def organization():
    """
    Organization fixture
    """
    return OrganizationFactory()


@pytest.mark.django_db
def test_route_control_exists(organization):
    """
    Test route control is created from
    create_route_control post_save signal
    """
    assert organization.route_controls is not None
    assert organization.route_controls.organization == organization


@pytest.mark.django_db
def test_route_distance_choices(organization):
    """
    Test Route avoidance choices throws ValidationError
    when the passed choice is not valid.
    """
    with pytest.raises(ValidationError, match="Value 'invalid' is not a valid choice."):
        organization.route_controls.mileage_unit = "invalid"
        organization.route_controls.full_clean()


@pytest.mark.django_db
def test_route_model_choices(organization):
    """
    Test Route model choices throws ValidationError
    when the passed choice is not a valid.
    """
    with pytest.raises(ValidationError, match="Value 'invalid' is not a valid choice."):
        organization.route_controls.traffic_model = "invalid"
        organization.route_controls.full_clean()
