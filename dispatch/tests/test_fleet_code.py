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

from dispatch.factories import FleetCodeFactory


@pytest.fixture
def fleet_code():
    """
    Fleet code fixture
    """
    yield FleetCodeFactory()


@pytest.mark.django_db
def test_fleet_code_creation(fleet_code):
    """
    Test fleet code creation
    """
    assert fleet_code is not None


@pytest.mark.django_db
def test_fleet_code_update(fleet_code):
    """
    Test fleet code update
    """
    fleet_code.code = "NEWC"
    fleet_code.save()
    assert fleet_code.code == "NEWC"
