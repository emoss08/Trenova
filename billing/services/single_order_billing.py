# --------------------------------------------------------------------------------------------------
#  COPYRIGHT(c) 2023 MONTA                                                                         -
#                                                                                                  -
#  This file is part of Monta.                                                                     -
#                                                                                                  -
#  The Monta software is licensed under the Business Source License 1.1. You are granted the right -
#  to copy, modify, and redistribute the software, but only for non-production use or with a total -
#  of less than three server instances. Starting from the Change Date (November 16, 2026), the     -
#  software will be made available under version 2 or later of the GNU General Public License.     -
#  If you use the software in violation of this license, your rights under the license will be     -
#  terminated automatically. The software is provided "as is," and the Licensor disclaims all      -
#  warranties and conditions. If you use this license's text or the "Business Source License" name -
#  and trademark, you must comply with the Licensor's covenants, which include specifying the      -
#  Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use     -
#  Grant, and not modifying the license in any other way.                                          -
# --------------------------------------------------------------------------------------------------

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
from utils.types import MODEL_UUID


def bill_order(*, user_id: MODEL_UUID, order: Order) -> None:
    """Bill a single order by performing multiple operations.

    This function checks if the billing control is valid, sets the billing requirements,
    sets the order document IDs, checks if the billing requirements are met, sets the order
    as billed, and sends a billing email if the requirements are met. If the billing control is
    not valid, or if the billing requirements are not met, an exception is created.

    Args:
        user_id (str): The ID of the user performing the billing actions.
        order (Order): The order to bill

    Returns:
        None: This function does not return anything.

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
