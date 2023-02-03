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

from billing.tests.factories import AccessorialChargeFactory


@pytest.fixture
def accessorial_charge() -> None:
    """
    Accessorial charge fixture
    """
    yield AccessorialChargeFactory()


@pytest.mark.django_db
def test_accessorial_charge_creation(accessorial_charge) -> None:
    """
    Test accessorial charge creation
    """
    assert accessorial_charge is not None


@pytest.mark.django_db
def test_accessorial_charge_update(accessorial_charge) -> None:
    """
    Test accessorial charge update
    """
    accessorial_charge.name = "New name"
    accessorial_charge.save()
    assert accessorial_charge.name == "New name"
