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


@pytest.fixture()
def fleet_code():
    """
    Fleet code fixture
    """
    return factories.FleetCodeFactory()


@pytest.fixture()
def comment_type():
    """
    Comment type fixture
    """
    return factories.CommentTypeFactory()


@pytest.fixture()
def dispatch_control():
    """
    Dispatch control fixture
    """
    return factories.DispatchControlFactory()


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


@pytest.mark.django_db
def test_comment_type_creation(comment_type):
    """
    Test comment type creation
    """
    assert comment_type is not None


@pytest.mark.django_db
def test_comment_type_update(comment_type):
    """
    Test comment type update
    """
    comment_type.code = "NEWC"
    comment_type.save()
    assert comment_type.code == "NEWC"
