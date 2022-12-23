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

from dispatch import factories


@pytest.fixture()
def delay_code():
    """
    Delay code fixture
    """
    return factories.DelayCodeFactory()


@pytest.mark.django_db
def test_delay_code_creation(delay_code):
    """
    Test delay code creation
    """
    assert delay_code is not None


@pytest.mark.django_db
def test_delay_code_update(delay_code):
    """
    Test delay code update
    """
    delay_code.code = "NEWC"
    delay_code.save()
    assert delay_code.code == "NEWC"
