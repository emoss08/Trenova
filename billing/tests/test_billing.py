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

import pytest
from django.core import mail
from django.core.exceptions import ValidationError

from accounts.models import User
from accounts.tests.factories import UserFactory
from billing import models, selectors, services
from billing.services import generate_invoice_number
from customer.factories import CustomerFactory
from customer.models import Customer
from order.models import Order
from order.tests.factories import OrderFactory
from organization.models import Organization
from utils.models import StatusChoices
from worker.models import Worker

pytestmark = pytest.mark.django_db


def test_generate_invoice_number(organization: Organization) -> None:
    """
    Test that invoice number increments by 1 for each new invoice
    and adds the correct suffix when an order is rebilled.
    """
    order_1 = OrderFactory()
    user = UserFactory()

    order_movements = order_1.movements.all()
    order_movements.update(status="C")

    order_1.status = "C"
    order_1.save()

    order_2 = OrderFactory()

    order_2_movements = order_2.movements.all()
    order_2_movements.update(status="C")

    order_2.status = "C"
    order_2.save()

    # Test first invoice
    invoice_1 = models.BillingQueue.objects.create(
        organization=organization, order=order_1, user=user, customer=order_1.customer
    )
    assert (
        invoice_1.invoice_number
        == f"{user.organization.invoice_control.invoice_number_prefix}00001"
    )

    # Test second invoice
    invoice_2 = models.BillingQueue.objects.create(
        organization=organization, order=order_2, user=user, customer=order_2.customer
    )
    assert (
        invoice_2.invoice_number
        == f"{user.organization.invoice_control.invoice_number_prefix}00002"
    )

    # Test rebilling first invoice (Credit Memo)
    invoice_1_cm = models.BillingQueue.objects.create(
        organization=organization,
        order=order_1,
        user=user,
        bill_type=models.BillingQueue.BillTypeChoices.CREDIT,
        customer=order_2.customer,
    )
    assert (
        invoice_1_cm.invoice_number
        == f"{user.organization.invoice_control.invoice_number_prefix}00001"
    )

    # Test rebilling first invoice again (New invoice with A suffix)
    invoice_1_a = models.BillingQueue.objects.create(
        organization=organization, order=order_1, user=user, customer=order_1.customer
    )
    assert (
        invoice_1_a.invoice_number
        == f"{user.organization.invoice_control.invoice_number_prefix}00001A"
    )

    # Test rebilling first invoice one more time (New invoice with B suffix)
    invoice_1_b = models.BillingQueue.objects.create(
        organization=organization, order=order_1, user=user, customer=order_1.customer
    )
    assert (
        invoice_1_b.invoice_number
        == f"{user.organization.invoice_control.invoice_number_prefix}00001B"
    )


def test_invoice_number_generation(
    organization: Organization, customer: Customer, user: User, worker: Worker
) -> None:
    """
    Test that invoice number is generated for each new invoice
    """
    order = OrderFactory()

    order_movements = order.movements.all()
    order_movements.update(status="C")

    order.status = "C"
    order.save()

    invoice = models.BillingQueue.objects.create(
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
    assert (
        invoice.invoice_number
        == f"{user.organization.invoice_control.invoice_number_prefix}00001"
    )


def test_invoice_number_increments(
    organization: Organization, customer: Customer, user: User, worker: Worker
) -> None:
    """
    Test that invoice number increments by 1 for each new invoice
    """
    order = OrderFactory()

    order_movements = order.movements.all()
    order_movements.update(status="C")

    order.status = "C"
    order.save()

    order_2 = OrderFactory()

    order_2_movements = order_2.movements.all()
    order_2_movements.update(status="C")

    order_2.status = "C"
    order_2.save()

    invoice = models.BillingQueue.objects.create(
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
    invoice.invoice_number = generate_invoice_number(instance=invoice)
    invoice.save()

    second_invoice = models.BillingQueue.objects.create(
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
    second_invoice.invoice_number = generate_invoice_number(instance=second_invoice)
    second_invoice.save()

    assert invoice.invoice_number is not None
    assert (
        invoice.invoice_number
        == f"{user.organization.invoice_control.invoice_number_prefix}00001"
    )
    assert second_invoice.invoice_number is not None
    assert (
        second_invoice.invoice_number
        == f"{user.organization.invoice_control.invoice_number_prefix}00002"
    )


def test_unbilled_order_in_billing_history(order: Order) -> None:
    """
    Test ValidationError is thrown when adding an order in billing history
    that hasn't billed.
    """

    with pytest.raises(ValidationError) as excinfo:
        models.BillingHistory.objects.create(
            organization=order.organization,
            order=order,
        )

    assert excinfo.value.message_dict["order"] == [
        "Order has not been billed. Please try again with a different order."
    ]


def test_billing_control_hook(organization: Organization) -> None:
    """
    Test that the billing control hook is created when a new organization is
    created.
    """
    assert organization.billing_control is not None


def test_auto_bill_criteria_required_when_auto_bill_true(
    organization: Organization,
) -> None:
    """
    Test if `auto_bill_orders` is true & `auto_bill_criteria` is blank that a
    `ValidationError` is thrown.
    """
    billing_control = organization.billing_control

    with pytest.raises(ValidationError) as excinfo:
        billing_control.auto_bill_orders = True
        billing_control.auto_bill_criteria = None
        billing_control.full_clean()

    assert excinfo.value.message_dict["auto_bill_criteria"] == [
        "Auto Billing criteria is required when `Auto Bill Orders` is on. Please try again."
    ]


def test_auto_bill_criteria_choices_is_invalid(organization: Organization) -> None:
    """
    Test when passing invalid choice to `auto_bill_criteria` that a
    `ValidationError` is thrown.
    """

    billing_control = organization.billing_control

    with pytest.raises(ValidationError) as excinfo:
        billing_control.auto_bill_criteria = "invalid"
        billing_control.full_clean()

    assert excinfo.value.message_dict["auto_bill_criteria"] == [
        "Value 'invalid' is not a valid choice."
    ]


def test_get_billable_orders_completed() -> None:
    """
    Test that get_billable_orders returns orders that are completed and not
    billed. When the billing_control.order_transfer_criteria is set to
    "COMPLETED".
    """
    # create an order that is ready to bill
    order = OrderFactory()

    order_movements = order.movements.all()
    order_movements.update(status="C")

    order.status = "C"
    order.billed = False
    order.transferred_to_billing = False
    order.billing_transfer_date = None
    order.save()

    # set the order_transfer_criteria on the organization's billing_control
    order.organization.billing_control.order_transfer_criteria = "COMPLETED"
    order.organization.billing_control.save()
    billable_orders = selectors.get_billable_orders(organization=order.organization)

    for order in billable_orders:
        assert order.status == "C"
        assert not order.billed
        assert not order.transferred_to_billing
        assert order.billing_transfer_date is None


def test_get_billable_orders_ready_and_completed() -> None:
    """
    Test that get_billable_orders returns orders that are completed and not
    billed. When the billing_control.order_transfer_criteria is set to
    "READY_AND_COMPLETED".
    """
    order = OrderFactory()

    order_movements = order.movements.all()
    order_movements.update(status="C")

    order.status = "C"
    order.billed = False
    order.transferred_to_billing = False
    order.billing_transfer_date = None
    order.ready_to_bill = True
    order.save()

    # set the order_transfer_criteria on the organization's billing_control
    order.organization.billing_control.order_transfer_criteria = "READY_AND_COMPLETED"
    order.organization.billing_control.save()
    billable_orders = selectors.get_billable_orders(organization=order.organization)

    for order in billable_orders:
        assert order.status == "C"
        assert not order.billed
        assert not order.transferred_to_billing
        assert order.billing_transfer_date is None


def test_get_billable_orders_ready() -> None:
    """
    Test that get_billable_orders returns orders that are completed and not
    billed. When the billing_control.order_transfer_criteria is set to
    "READY_AND_COMPLETED".
    """
    # create an order that is ready to bill
    order = OrderFactory()

    order_movements = order.movements.all()
    order_movements.update(status="C")

    order.status = "C"
    order.billed = False
    order.transferred_to_billing = False
    order.billing_transfer_date = None
    order.ready_to_bill = True
    order.save()

    # set the order_transfer_criteria on the organization's billing_control
    order.organization.billing_control.order_transfer_criteria = "READY_TO_BILL"
    order.organization.billing_control.save()
    billable_orders = selectors.get_billable_orders(organization=order.organization)

    for order in billable_orders:
        assert order.status == "C"
        assert not order.billed
        assert not order.transferred_to_billing
        assert order.billing_transfer_date is None


def test_get_billing_queue_information() -> None:
    """
    Test that the correct billing queue is returned when using the
    `get_billing_queue_information` selector.
    """

    customer = CustomerFactory()

    order = OrderFactory()

    order_movements = order.movements.all()
    order_movements.update(status="C")

    order.ready_to_bill = True
    order.status = "C"
    order.save()

    billing_queue = models.BillingQueue.objects.create(
        organization=order.organization, order=order, customer=customer
    )

    result = selectors.get_billing_queue_information(order=order)
    assert result == billing_queue


def test_cannot_delete_billing_history(
    organization: Organization, order: Order
) -> None:
    """
    Test that if the organization has remove_billing_history as false that
    the billing history cannot be deleted.
    """
    organization.billing_control.remove_billing_history = False
    organization.billing_control.save()

    order.billed = True
    order.save()

    billing_history = models.BillingHistory.objects.create(
        organization=organization, order=order, customer=order.customer
    )

    with pytest.raises(ValidationError) as excinfo:
        billing_history.delete()

    assert excinfo.value.message_dict["organization"] == [
        "Billing history cannot be deleted. Please try again."
    ]


def test_can_delete_billing_history(organization: Organization, order: Order) -> None:
    """
    Test that if the organization has remove_billing_history as true that the billing
    history can be deleted.
    """
    organization.billing_control.remove_billing_history = True
    organization.billing_control.save()

    order.billed = True
    order.save()

    billing_history = models.BillingHistory.objects.create(
        organization=organization, order=order, customer=order.customer
    )

    billing_history.delete()

    assert models.BillingHistory.objects.count() == 0


def test_generate_invoice_number_before_save(order: Order) -> None:
    """
    Test that the invoice number is generated before the save method is called.
    """

    order.status = StatusChoices.COMPLETED
    order.ready_to_bill = True

    billing_queue = models.BillingQueue.objects.create(
        organization=order.organization, order=order, customer=order.customer
    )

    assert (
        billing_queue.invoice_number
        == f"{order.organization.invoice_control.invoice_number_prefix}00001"
    )


def test_save_order_details_to_billing_history_before_save(order: Order) -> None:
    """
    Test that the order details are saved to the billing history before the
    save method is called.
    """
    order.billed = True
    order.save()

    billing_history = models.BillingHistory.objects.create(
        organization=order.organization, order=order, customer=order.customer
    )

    assert billing_history.pieces == order.pieces
    assert billing_history.order_type == order.order_type
    assert billing_history.weight == order.weight
    assert billing_history.mileage == order.mileage
    assert billing_history.revenue_code == order.revenue_code
    assert billing_history.commodity == order.commodity
    assert billing_history.bol_number == order.bol_number
    assert billing_history.customer == order.customer
    assert billing_history.other_charge_total == order.other_charge_amount
    assert billing_history.freight_charge_amount == order.freight_charge_amount
    assert billing_history.total_amount == order.sub_total


def test_transfer_order_to_billing_queue(organization: Organization) -> None:
    """
    Test an order is transferred to the billing queue.
    """

    # set the order_transfer_criteria on the organization's billing_control
    organization.billing_control.order_transfer_criteria = "READY_TO_BILL"
    organization.billing_control.save()

    order = OrderFactory(organization=organization)

    order_movements = order.movements.all()
    order_movements.update(status="C")

    order.status = "C"
    order.ready_to_bill = True
    order.transferred_to_billing = False
    order.billing_transfer_date = None
    order.save()

    user = UserFactory(organization=organization)

    services.transfer_to_billing_queue_service(
        user_id=user.id,
        order_pros=[order.pro_number],
        task_id=str(uuid.uuid4()),
    )

    billing_queue = models.BillingQueue.objects.get(order=order)
    billing_transfer_log = models.BillingTransferLog.objects.get(order=order)

    order.refresh_from_db()

    assert order.transferred_to_billing
    assert order.billing_transfer_date is not None
    assert billing_queue.order_type == order.order_type
    assert billing_queue.weight == order.weight
    assert billing_queue.pieces == order.pieces
    assert billing_queue.revenue_code == order.revenue_code
    assert billing_queue.commodity == order.commodity
    assert billing_queue.bol_number == order.bol_number
    assert billing_queue.customer == order.customer
    assert billing_queue.bill_type == "INVOICE"

    # Check that the billing_transfer_log was created
    assert billing_transfer_log


def test_bill_orders(
    organization: Organization,
    user: User,
    worker: Worker,
) -> None:
    """
    Test that the orders are billed correctly.
    """

    # set the order_transfer_criteria on the organization's billing_control
    organization.billing_control.order_transfer_criteria = "READY_TO_BILL"
    organization.billing_control.save()

    # Create an order from the Order Factory
    order = OrderFactory(organization=organization)

    # Update the order movements to be completed
    order_movements = order.movements.all()
    order_movements.update(status="C")

    # Update the order to be ready to bill
    order.status = "C"
    order.ready_to_bill = True
    order.transferred_to_billing = False
    order.billing_transfer_date = None
    order.save()

    # Create a User from the User Factory
    user = UserFactory(organization=organization)

    # transfer the order to the billing queue
    services.transfer_to_billing_queue_service(
        user_id=user.id,
        order_pros=[order.pro_number],
        task_id=str(uuid.uuid4()),
    )

    # Bill all the orders, in the billing queue.
    invoices = models.BillingQueue.objects.all()
    services.bill_orders(user_id=user.id, invoices=invoices)

    # Query the billing history to make sure it was created.
    billing_history = models.BillingHistory.objects.get(order=order)
    billing_history.refresh_from_db()

    assert billing_history.order == order
    assert billing_history.organization == order.organization
    assert billing_history.order_type == order.order_type
    assert billing_history.revenue_code == order.revenue_code
    assert billing_history.customer == order.customer
    assert billing_history.commodity == order.commodity
    assert billing_history.bol_number == order.bol_number
    assert (
        billing_history.invoice_number
        == f"{user.organization.invoice_control.invoice_number_prefix}00001"
    )

    order.refresh_from_db()
    assert order.billed is True
    assert order.bill_date is not None
    assert mail.outbox[0].subject == f"New invoice from {user.organization.name}"
    assert (
        mail.outbox[0].body
        == f"Please see attached invoice for invoice: {order.pro_number}"
    )


def test_single_order_billing_service(
    organization: Organization,
    user: User,
    worker: Worker,
) -> None:
    """
    Test an single order can be billed.
    """
    # set the order_transfer_criteria on the organization's billing_control
    organization.billing_control.order_transfer_criteria = "READY_TO_BILL"
    organization.billing_control.save()

    # Create an order from the Order Factory
    order = OrderFactory(organization=organization)

    # Update the order movements to be completed
    order_movements = order.movements.all()
    order_movements.update(status="C")

    # Update the order to be ready to bill
    order.status = "C"
    order.ready_to_bill = True
    order.transferred_to_billing = False
    order.billing_transfer_date = None
    order.save()

    # Create a User from the User Factory
    user = UserFactory(organization=organization)

    # transfer the order to the billing queue
    services.transfer_to_billing_queue_service(
        user_id=user.id,
        order_pros=[order.pro_number],
        task_id=str(uuid.uuid4()),
    )

    # Bill all the orders, in the billing queue.
    invoice = models.BillingQueue.objects.get(order=order)
    services.bill_orders(user_id=user.id, invoices=invoice)

    # Query the billing history to make sure it was created.
    billing_history = models.BillingHistory.objects.get(order=order)
    billing_history.refresh_from_db()

    assert billing_history.order == order
    assert billing_history.organization == order.organization
    assert billing_history.order_type == order.order_type
    assert billing_history.revenue_code == order.revenue_code
    assert billing_history.customer == order.customer
    assert billing_history.commodity == order.commodity
    assert billing_history.bol_number == order.bol_number
    assert (
        billing_history.invoice_number
        == f"{user.organization.invoice_control.invoice_number_prefix}00001"
    )

    order.refresh_from_db()
    assert order.billed is True
    assert order.bill_date is not None
    assert mail.outbox[0].subject == f"New invoice from {user.organization.name}"
    assert (
        mail.outbox[0].body
        == f"Please see attached invoice for invoice: {order.pro_number}"
    )
