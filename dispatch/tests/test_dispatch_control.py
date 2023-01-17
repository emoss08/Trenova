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
def test_dispatch_control_creation(organization):
    """
    Test dispatch control is created from
    create_dispatch_control post_save signal
    """
    assert organization.dispatch_control.driver_assign is True
    assert organization.dispatch_control.organization == organization


@pytest.mark.django_db
def test_service_incident_control_choices(organization):
    """
    Test Service incident control choices throws ValidationError
    when the passed choice is not valid.
    """
    with pytest.raises(ValidationError) as excinfo:
        organization.dispatch_control.record_service_incident = "invalid"
        organization.dispatch_control.full_clean()

    assert excinfo.value.message_dict["record_service_incident"] == [
        "Value 'invalid' is not a valid choice."
    ]


@pytest.mark.django_db
def test_distance_method_choices(organization):
    """
    Test Service incident control choices throws ValidationError
    when the passed choice is not valid.
    """
    with pytest.raises(ValidationError) as excinfo:
        organization.dispatch_control.distance_method = "invalid"
        organization.dispatch_control.full_clean()

    assert excinfo.value.message_dict["distance_method"] == [
        "Value 'invalid' is not a valid choice."
    ]


@pytest.mark.django_db
def test_dispatch_control_google_integration(organization):
    """
    Test Service incident control choices throws ValidationError
    when the passed choice is not valid.
    """
    with pytest.raises(
        ValidationError) as excinfo:
        organization.dispatch_control.distance_method = "Google"
        organization.dispatch_control.full_clean()

    assert excinfo.value.message_dict["distance_method"] == [
        "Google Maps integration is not configured for the organization."
        " Please configure the integration before selecting Google as "
        "the distance method."
    ]
