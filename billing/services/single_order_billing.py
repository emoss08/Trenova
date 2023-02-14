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
from django.shortcuts import get_object_or_404

from accounts.models import User
from billing.services.billing_service import (
    check_billing_control,
    check_billing_requirements,
    create_billing_exception,
    send_billing_email,
    set_billing_requirements,
    set_order_billed,
    set_order_documents,
)
from order.models import Order


def bill_order(*, user_id: str, order: Order) -> None:
    """Bill a single order by performing multiple operations.

    This function checks if the billing control is valid, sets the billing requirements,
    sets the order document IDs, checks if the billing requirements are met, sets the order
    as billed, and sends a billing email if the requirements are met. If the billing control is
    not valid, or if the billing requirements are not met, an exception is created.

    Args:
        user_id (str): The ID of the user performing the billing actions.
        order (Order): The order to bill

    Returns:
        None: None

    Raises:
        None
    """

    user: User = get_object_or_404(User, id=user_id)

    if check_billing_control(user=user):
        set_billing_requirements(customer=order.customer)
        set_order_documents(order=order)
        if check_billing_requirements(user=user, order=order):
            set_order_billed(order=order)
            send_billing_email(user=user, order=order)
        else:
            create_billing_exception(
                user=user,
                exception_type="PAPERWORK",
                order=order,
                exception_message="Billing requirement not met",
            )
    else:
        set_order_billed(order=order)
