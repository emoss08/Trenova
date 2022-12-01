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

from organization.factories import DepotFactory


@pytest.fixture()
def depot():
    """
    Depot fixture
    """
    return DepotFactory()


@pytest.mark.django_db
def test_depot_creation(depot):
    """
    Test depot creation
    """
    assert depot is not None


@pytest.mark.django_db
def test_depot_update(depot):
    """
    Test depot update
    """
    depot.name = "New Name"
    depot.save()
    assert depot.name == "New Name"


@pytest.mark.django_db
def test_depot_organization(depot):
    """
    Test dispatch control is created from
    create_depot_detail post_save signal
    """
    assert depot.depot_details.organization == depot.organization
