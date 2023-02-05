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
from django.core import mail
from django.core.exceptions import ValidationError
from django.test import RequestFactory
from django.utils import timezone

from billing.models import BillingHistory, BillingQueue
from billing.services.order_billing import BillingService
from order.tests.factories import OrderFactory

pytestmark = pytest.mark.django_db


def test_bill_orders(
    organization,
    customer,
    user,
    worker,
) -> None:
    order = OrderFactory(status="C")
    BillingQueue.objects.create(
        organization=user.organization,
        order_type=order.order_type,
        order=order,
        revenue_code=order.revenue_code,
        customer=customer,
        worker=worker,
        commodity=order.commodity,
        bol_number=order.bol_number,
        user=user,
    )
    request = RequestFactory().get("/")
    request.user = user

    BillingService(request=request).bill_orders()

    billing_queue = BillingQueue.objects.all()
    billing_history = BillingHistory.objects.get(order=order)

    assert billing_queue.count() == 0
    assert billing_history.order == order
    assert billing_history.organization == order.organization
    assert billing_history.order_type == order.order_type
    assert billing_history.revenue_code == order.revenue_code
    assert billing_history.customer == order.customer
    assert billing_history.commodity == order.commodity
    assert billing_history.bol_number == order.bol_number

    order.refresh_from_db()
    assert order.billed is True
    assert order.bill_date == timezone.now().date()
    assert mail.outbox[0].subject == f"New invoice from {user.organization.name}"
    assert (
        mail.outbox[0].body
        == f"Please see attached invoice for invoice: {order.pro_number}"
    )


def test_invoice_number_generation(organization, customer, user, worker) -> None:
    """
    Test that invoice number is generated for each new invoice
    """
    order = OrderFactory(status="C")
    invoice = BillingQueue.objects.create(
        organization=user.organization,
        order_type=order.order_type,
        order=order,
        revenue_code=order.revenue_code,
        customer=customer,
        worker=worker,
        commodity=order.commodity,
        bol_number=order.bol_number,
        user=user,
    )
    assert invoice.invoice_number is not None
    assert invoice.invoice_number == f"{user.organization.scac_code}00001"

def test_invoice_number_increments(organization, customer, user, worker) -> None:
    """
    Test that invoice number increments by 1 for each new invoice
    """
    order = OrderFactory(status="C")
    order_2 = OrderFactory(status="C")
    invoice = BillingQueue.objects.create(
        organization=user.organization,
        order_type=order.order_type,
        order=order,
        revenue_code=order.revenue_code,
        customer=customer,
        worker=worker,
        commodity=order.commodity,
        bol_number=order.bol_number,
        user=user,
    )
    second_invoice = BillingQueue.objects.create(
        organization=user.organization,
        order_type=order_2.order_type,
        order=order_2,
        revenue_code=order_2.revenue_code,
        customer=customer,
        worker=worker,
        commodity=order_2.commodity,
        bol_number=order_2.bol_number,
        user=user,
    )

    assert invoice.invoice_number is not None
    assert invoice.invoice_number == f"{user.organization.scac_code}00001"
    assert second_invoice.invoice_number is not None
    assert second_invoice.invoice_number == f"{user.organization.scac_code}00002"

def test_unbilled_order_in_billing_history(order) -> None:
    """
    Test ValidationError is thrown when adding an order in billing history
    that hasn't billed.
    """

    with pytest.raises(ValidationError) as excinfo:
        BillingHistory.objects.create(
            organization=order.organization,
            order=order,
        )

    assert excinfo.value.message_dict["order"] == [
        "Order has not been billed. Please try again with a different order."
    ]
