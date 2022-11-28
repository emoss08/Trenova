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

from order.factories import OrderTypeFactory


@pytest.fixture()
def order_type():
    """
    OrderType fixture
    """
    return OrderTypeFactory()


@pytest.mark.django_db
def test_order_type_creation(order_type):
    """
    Test order type creation
    """
    assert order_type is not None


@pytest.mark.django_db
def test_order_type_update(order_type):
    """
    Test order type update
    """
    order_type.name = "New name"
    order_type.save()
    assert order_type.name == "New name"
