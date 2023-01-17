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

from django.db.models import Sum

from order.models import Order
from stops.models import Stop


def total_piece_count_for_order(*, order: Order) -> int:
    """Return the total piece count for an order

    Args:
        order (Order): Order instance

    Returns:
        int: Total piece count for an order
    """
    return Stop.objects.filter(movement__order__exact=order).aggregate(Sum('piece_count'))['piece_count__sum']

def total_weight_for_order(*, order: Order) -> int:
    """Return the total weight for an order

    Args:
        order (Order): Order instance

    Returns:
        int: Total weight for an order
    """
    return Stop.objects.filter(movement__order__exact=order).aggregate(
        Sum("weight")
    )["weight__sum"]
