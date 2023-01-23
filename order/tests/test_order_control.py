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

from order import models

pytestmark = pytest.mark.django_db


class TestOrderControl:
    """
    Class to test Order Control
    """

    def test_list(self, organization):
        """
        Test Order Control is created when the organization is
        created. This process happens via signals.
        """

        assert organization.order_control is not None

    def test_update(self, organization):
        """
        Test order type update
        """

        order_control = models.OrderControl.objects.get(organization=organization)

        order_control.auto_rate_orders = False
        order_control.calculate_distance = True
        order_control.enforce_customer = True
        order_control.enforce_rev_code = True

        order_control.save()

        assert order_control is not None
        assert not order_control.auto_rate_orders
        assert order_control.calculate_distance
        assert order_control.enforce_customer
        assert order_control.enforce_rev_code
