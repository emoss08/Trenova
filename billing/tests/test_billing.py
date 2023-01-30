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
from django.test import RequestFactory
from django.utils import timezone

from billing.models import BillingQueue
from billing.services import BillingService
from customer.factories import CustomerBillingProfileFactory
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

    BillingService(request=request)

    billing_queue = BillingQueue.objects.all()
    assert billing_queue.count() == 0
    order.refresh_from_db()
    assert order.billed is True
    assert order.bill_date == timezone.now().date()
    assert mail.outbox[0].subject == f"New invoice from {user.organization.name}"
    assert (
        mail.outbox[0].body
        == f"Please see attached invoice for invoice: {order.pro_number}"
    )
