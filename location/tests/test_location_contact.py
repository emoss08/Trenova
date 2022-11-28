# Create your tests here.
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

from location.factories import (
    LocationContactFactory,
)


@pytest.fixture()
def location_contact():
    """
    Location fixture
    """
    return LocationContactFactory()


@pytest.mark.django_db
def test_add_contact_to_location(location_contact):
    """
    Test add contact to location
    """
    location = LocationContactFactory()
    location.location_contact = location_contact
    location.save()
    assert location.location_contact == location_contact


@pytest.mark.django_db
def test_update_contact_on_location(location_contact):
    """
    Test update contact to location
    """
    location = LocationContactFactory()
    location.location_contact = location_contact
    location.save()
    location.location_contact.name = "New name"
    location.save()
    assert location.location_contact.name == "New name"
