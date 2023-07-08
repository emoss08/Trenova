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
from typing import TYPE_CHECKING

from channels.db import database_sync_to_async
from django.db.models import Q, QuerySet
from notifications.signals import notify

from billing import models
from order.serializers import OrderSerializer
from utils.models import StatusChoices

if TYPE_CHECKING:
    from accounts.models import User
    from order.models import Order
    from organization.models import Organization
    from utils.types import ModelUUID


def get_billable_orders(
    *, organization: "Organization", order_pros: list[str] | None = None
) -> QuerySet["Order"] | None:
    """Retrieve billable orders for a given organization based on specified criteria.

    Args:
        organization (Organization): The organization for which to retrieve billable orders.
        order_pros (List[str] | None, optional): A list of order PRO numbers to filter by, if specified.
            Defaults to None.

    Returns:
        QuerySet[Order] | None: A queryset of billable orders, or None if no billable orders are found.
    """

    # Map BillingControl.OrderTransferCriteriaChoices to the corresponding query
    criteria_to_query = {
        models.BillingControl.OrderTransferCriteriaChoices.READY_AND_COMPLETED: Q(
            status=StatusChoices.COMPLETED
        )
        & Q(ready_to_bill=True),
        models.BillingControl.OrderTransferCriteriaChoices.COMPLETED: Q(
            status=StatusChoices.COMPLETED
        ),
        models.BillingControl.OrderTransferCriteriaChoices.READY_TO_BILL: Q(
            ready_to_bill=True
        ),
    }

    query: Q = (
        Q(billed=False)
        & Q(transferred_to_billing=False)
        & Q(billing_transfer_date__isnull=True)
    )
    order_criteria_query: Q | None = criteria_to_query.get(
        organization.billing_control.order_transfer_criteria
    )

    if order_criteria_query is not None:
        query &= order_criteria_query

    if order_pros:
        query &= Q(pro_number__in=order_pros)

    orders = organization.orders.filter(query)

    return orders if orders.exists() else None


def get_billing_queue_information(*, order: "Order") -> models.BillingQueue | None:
    """Retrieve the most recent billing queue information for a given order.

    Args:
        order (Order): The order for which to retrieve billing queue information.

    Returns:
        models.BillingQueue | None: The most recent BillingQueue instance for the given order,
            or None if no billing queue information is found.
    """
    return models.BillingQueue.objects.filter(order=order).last()


def get_billing_queue(
    *, user: "User", task_id: str | uuid.UUID
) -> QuerySet[models.BillingQueue]:
    """Retrieve the billing queue for a given user's organization.

    Args:
        user (User): The user whose organization's billing queue should be retrieved.
        task_id (str | uuid.UUID): The ID of the task that initiated the retrieval.

    Returns:
        QuerySet[models.BillingQueue]: A queryset of BillingQueue instances for the user's organization.
    """
    billing_queue = models.BillingQueue.objects.filter(organization=user.organization)
    if not billing_queue:
        notify.send(
            user,
            organization=user.organization,
            recipient=user,
            level="info",
            verb="Order Billing Exception",
            description=f"No Orders in the billing queue for task: {task_id}",
        )
    return billing_queue


def get_invoice_by_id(*, invoice_id: "ModelUUID") -> models.BillingQueue | None:
    """Retrieve a BillingQueue instance by its invoice ID.

    Args:
        invoice_id (ModelUUID): The ID of the invoice to retrieve.

    Returns:
        models.BillingQueue | None: The BillingQueue instance with the specified invoice ID,
            or None if the invoice is not found.
    """
    try:
        return models.BillingQueue.objects.get(pk__exact=invoice_id)
    except models.BillingQueue.DoesNotExist:
        return None


def get_invoices_by_invoice_number(
    *, invoices: list[str]
) -> QuerySet[models.BillingQueue]:
    """Retrieves a queryset of BillingQueue objects by their invoice numbers.

    Args:
        invoices (list[str]):

    Returns:
        QuerySet[models.BillingQueue]: A queryset of BillingQueue objects.
    """
    return models.BillingQueue.objects.filter(invoice_number__in=invoices)
