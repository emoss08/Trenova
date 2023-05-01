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
import uuid
from collections.abc import Iterable
from typing import TYPE_CHECKING, List

from django.core.exceptions import ValidationError
from django.db import IntegrityError
from django.shortcuts import get_object_or_404
from django.utils import timezone

from accounts.models import User
from billing import exceptions, models, selectors, utils
from order.models import Order

if TYPE_CHECKING:
    from utils.types import ModelUUID


def generate_invoice_number(*, instance: models.BillingQueue) -> str:
    """Generate a new invoice number for a BillingQueue instance.

    Args:
        instance (models.BillingQueue): The BillingQueue instance to generate an invoice number for.

    Returns:
        str: The generated invoice number.
    """

    if not instance.invoice_number:
        if (
            latest_invoice := models.BillingQueue.objects.only("invoice_number")
            .order_by("invoice_number")
            .last()
        ):
            latest_invoice_number = int(
                latest_invoice.invoice_number.split(
                    instance.organization.invoice_control.invoice_number_prefix
                )[-1]
            )
            instance.invoice_number = "{}{:05d}".format(
                instance.organization.invoice_control.invoice_number_prefix,
                latest_invoice_number + 1,
            )
        else:
            instance.invoice_number = (
                f"{instance.organization.invoice_control.invoice_number_prefix}00001"
            )

    return instance.invoice_number


def transfer_to_billing_queue_service(
    *, user_id: "ModelUUID", order_pros: list[str], task_id: str
) -> str:
    """Transfer eligible orders to the billing queue.

    Args:
        user_id (ModelUUID): The ID of the user transferring the orders.
        order_pros (List[str]): A list of order PRO numbers to be transferred.
        task_id (str): The ID of the task that initiated the transfer.

    Returns:
        str: A message indicating the success of the transfer and the number of orders transferred.

    Raises:
        exceptions.BillingException: If no eligible orders are found for transfer.
    """

    # Get the user
    user = get_object_or_404(User, id=user_id)

    billing_control = user.organization.billing_control

    # Get the billable orders
    orders = selectors.get_billable_orders(
        organization=user.organization, order_pros=order_pros
    )

    # If there are no orders, raise an BillingException
    if not orders:
        raise exceptions.BillingException(
            f"No orders found to be eligible for transfer. Orders must be marked {billing_control.order_transfer_criteria}"
        )

    # Get the current time
    now = timezone.now()

    # Create a list of BillingTransferLog objects
    transfer_log = []

    # Loop through the orders and create a BillingQueue object for each
    for order in orders:
        try:
            # Create a BillingQueue object
            models.BillingQueue.objects.create(
                organization=order.organization, order=order, customer=order.customer
            )

            # Update the order
            order.transferred_to_billing = True
            order.billing_transfer_date = now

            # Create a BillingTransferLog object
            transfer_log.append(
                models.BillingTransferLog(
                    order=order,
                    organization=order.organization,
                    task_id=task_id,
                    transferred_at=now,
                    transferred_by=user,
                )
            )

        # If there is a ValidationError or IntegrityError, create a BillingException
        except* ValidationError as validation_error:
            utils.create_billing_exception(
                user=user,
                exception_type="OTHER",
                invoice=order,
                exception_message=f"Order {order.pro_number} failed to transfer to billing queue: {validation_error}",
            )
        except* IntegrityError as integrity_error:
            utils.create_billing_exception(
                user=user,
                exception_type="OTHER",
                invoice=order,
                exception_message=f"Order {order.pro_number} failed to transfer to billing queue: {integrity_error}",
            )

    # Bulk update the orders
    Order.objects.bulk_update(
        orders, ["transferred_to_billing", "billing_transfer_date"]
    )

    # Bulk create the transfer log
    models.BillingTransferLog.objects.bulk_create(transfer_log)

    # Return a success message
    return f"Successfully transferred {len(orders)} orders to billing queue."


def mass_order_billing_service(
    *, user_id: "ModelUUID", task_id: str | uuid.UUID
) -> None:
    """Process the billing for multiple orders.

    Args:
        user_id (ModelUUID): The ID of the user initiating the mass billing.
        task_id (str | uuid.UUID): The ID of the task that initiated the mass billing.

    Returns:
        None: This function does not return anything.
    """

    user: User = get_object_or_404(User, id=user_id)
    orders = selectors.get_billing_queue(user=user, task_id=task_id)
    bill_orders(user_id=user_id, invoices=orders)


def bill_orders(
    *,
    user_id: "ModelUUID",
    invoices: Iterable[models.BillingQueue] | models.BillingQueue,
) -> None:
    """Bill the specified orders.

    Args:
        user_id (ModelUUID): The ID of the user responsible for billing the orders.
        invoices (Iterable[models.BillingQueue] | models.BillingQueue): An iterable of BillingQueue instances
            or a single BillingQueue instance representing the orders to be billed.

    Returns:
        None: This function does not return anything.
    """

    user = get_object_or_404(User, id=user_id)

    # If invoices is a BillingQueue object, convert it to a list
    if isinstance(invoices, models.BillingQueue):
        invoices = [invoices]

    # Check the organization enforces customer billing_requirements
    organization_enforces_billing = utils.check_organization_enforces_customer_billing(
        organization=user.organization
    )

    # Loop through the invoices and bill them
    for invoice in invoices:
        # If the organization enforces customer billing requirements, check them
        if organization_enforces_billing and not utils.check_billing_requirements(
            user=user, invoice=invoice
        ):
            # If the customer billing requirements are not met, create a BillingException
            utils.create_billing_exception(
                user=user,
                exception_type="PAPERWORK",
                invoice=invoice,
                exception_message="Billing requirement not met",
            )
        else:
            # If the customer billing requirements are met or not enforced, bill the order
            utils.order_billing_actions(invoice=invoice, user=user)
