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
from django.urls import reverse
from rest_framework.test import APIClient

from accounts.models import User
from accounts.tests.factories import UserFactory
from billing import models, selectors, services
from billing.services import generate_invoice_number
from billing.tests.factories import BillingQueueFactory
from customer.factories import CustomerFactory
from customer.models import Customer
from organization.models import BusinessUnit, Organization
from shipment.tests.factories import shipmentFactory
from utils.models import StatusChoices
from worker.models import Worker

pytestmark = pytest.mark.django_db


def test_generate_invoice_number(
    organization: Organization, business_unit: BusinessUnit
) -> None:
    """
    Test that invoice number increments by 1 for each new invoice
    and adds the correct suffix when an shipment is rebilled.
    """
    shipment_1 = shipmentFactory()
    user = UserFactory()

    shipment_movements = shipment_1.movements.all()
    shipment_movements.update(status="C")

    shipment_1.status = "C"
    shipment_1.save()

    # Test first invoice
    invoice_1 = models.BillingQueue.objects.create(
        organization=organization,
        business_unit=business_unit,
        shipment=shipment_1,
        user=user,
        customer=shipment_1.customer,
    )
    assert (
        invoice_1.invoice_number
        == f"{user.organization.invoice_control.invoice_number_prefix}{invoice_1.shipment.pro_number}".replace(
            "ORD", ""
        )
    )

    invoice_1_cm = models.BillingQueue.objects.create(
        organization=organization,
        business_unit=business_unit,
        shipment=shipment_1,
        user=user,
        bill_type=models.BillingQueue.BillTypeChoices.CREDIT,
        customer=shipment_1.customer,
    )
    assert (
        invoice_1_cm.invoice_number
        == f"{user.organization.invoice_control.invoice_number_prefix}{invoice_1.shipment.pro_number}".replace(
            "ORD", ""
        )
    )

    invoice_1_next_invoice = models.BillingQueue.objects.create(
        organization=organization,
        business_unit=business_unit,
        shipment=shipment_1,
        user=user,
        customer=shipment_1.customer,
    )

    assert (
        invoice_1_next_invoice.invoice_number
        == f"{user.organization.invoice_control.invoice_number_prefix}{invoice_1.shipment.pro_number}A".replace(
            "ORD", ""
        )
    )

    invoice_1_next_invoice_cm = models.BillingQueue.objects.create(
        organization=organization,
        business_unit=business_unit,
        shipment=shipment_1,
        user=user,
        bill_type=models.BillingQueue.BillTypeChoices.CREDIT,
        customer=shipment_1.customer,
    )

    assert (
        invoice_1_next_invoice_cm.invoice_number
        == f"{user.organization.invoice_control.invoice_number_prefix}{invoice_1.shipment.pro_number}A".replace(
            "ORD", ""
        )
    )

    invoice_1_final_invoice = models.BillingQueue.objects.create(
        organization=organization,
        shipment=shipment_1,
        user=user,
        business_unit=business_unit,
        customer=shipment_1.customer,
    )

    assert (
        invoice_1_final_invoice.invoice_number
        == f"{user.organization.invoice_control.invoice_number_prefix}{invoice_1.shipment.pro_number}B".replace(
            "ORD", ""
        )
    )

    invoice_1_final_invoice_cm = models.BillingQueue.objects.create(
        organization=organization,
        business_unit=business_unit,
        shipment=shipment_1,
        user=user,
        bill_type=models.BillingQueue.BillTypeChoices.CREDIT,
        customer=shipment_1.customer,
    )

    assert (
        invoice_1_final_invoice_cm.invoice_number
        == f"{user.organization.invoice_control.invoice_number_prefix}{invoice_1.shipment.pro_number}B".replace(
            "ORD", ""
        )
    )


def test_invoice_number_generation(
    organization: Organization,
    customer: Customer,
    user: User,
    worker: Worker,
    business_unit: BusinessUnit,
) -> None:
    """
    Test that invoice number is generated for each new invoice
    """
    shipment = shipmentFactory()

    shipment_movements = shipment.movements.all()
    shipment_movements.update(status="C")

    shipment.status = "C"
    shipment.save()

    invoice = models.BillingQueue.objects.create(
        organization=organization,
        business_unit=business_unit,
        shipment_type=shipment.shipment_type,
        shipment=shipment,
        revenue_code=shipment.revenue_code,
        customer=customer,
        worker=worker,
        commodity=shipment.commodity,
        bol_number=shipment.bol_number,
        user=user,
    )

    assert invoice.invoice_number is not None
    assert (
        invoice.invoice_number
        == f"{user.organization.invoice_control.invoice_number_prefix}{invoice.shipment.pro_number}".replace(
            "ORD", ""
        )
    )


def test_invoice_number_increments(
    organization: Organization,
    business_unit: BusinessUnit,
    customer: Customer,
    user: User,
    worker: Worker,
) -> None:
    """
    Test that invoice number increments by 1 for each new invoice
    """
    shipment = shipmentFactory()

    shipment_movements = shipment.movements.all()
    shipment_movements.update(status="C")

    shipment.status = "C"
    shipment.save()

    shipment_2 = shipmentFactory()

    shipment_2_movements = shipment_2.movements.all()
    shipment_2_movements.update(status="C")

    shipment_2.status = "C"
    shipment_2.save()

    invoice = models.BillingQueue.objects.create(
        organization=organization,
        business_unit=business_unit,
        shipment_type=shipment.shipment_type,
        shipment=shipment,
        revenue_code=shipment.revenue_code,
        customer=customer,
        worker=worker,
        commodity=shipment.commodity,
        bol_number=shipment.bol_number,
        user=user,
    )
    invoice.invoice_number = generate_invoice_number(instance=invoice)
    invoice.save()

    second_invoice = models.BillingQueue.objects.create(
        business_unit=business_unit,
        organization=user.organization,
        shipment_type=shipment_2.shipment_type,
        shipment=shipment_2,
        revenue_code=shipment_2.revenue_code,
        customer=customer,
        worker=worker,
        commodity=shipment_2.commodity,
        bol_number=shipment_2.bol_number,
        user=user,
    )
    second_invoice.invoice_number = generate_invoice_number(instance=second_invoice)
    second_invoice.save()

    assert invoice.invoice_number is not None
    assert (
        invoice.invoice_number
        == f"{user.organization.invoice_control.invoice_number_prefix}{invoice.shipment.pro_number}A".replace(
            "ORD", ""
        )
    )
    assert second_invoice.invoice_number is not None
    assert (
        second_invoice.invoice_number
        == f"{user.organization.invoice_control.invoice_number_prefix}{second_invoice.shipment.pro_number}A".replace(
            "ORD", ""
        )
    )


def test_unbilled_shipments_in_billing_history(
    shipment: Shipment, business_unit: BusinessUnit, organization: Organization
) -> None:
    """
    Test ValidationError is thrown when adding an shipment in billing history
    that hasn't billed.
    """

    with pytest.raises(ValidationError) as excinfo:
        models.BillingHistory.objects.create(
            business_unit=business_unit,
            organization=organization,
            shipment=shipment,
        )

    assert excinfo.value.message_dict["shipment"] == [
        "shipment has not been billed. Please try again with a different shipment."
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
    Test if `auto_bill_shipments` is true & `auto_bill_criteria` is blank that a
    `ValidationError` is thrown.
    """
    billing_control = organization.billing_control

    with pytest.raises(ValidationError) as excinfo:
        billing_control.auto_bill_shipments = True
        billing_control.auto_bill_criteria = None
        billing_control.full_clean()

    assert excinfo.value.message_dict["auto_bill_criteria"] == [
        "Auto Billing criteria is required when `Auto Bill shipments` is on. Please try again."
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


def test_get_billable_shipments_completed() -> None:
    """
    Test that get_billable_shipments returns shipments that are completed and not
    billed. When the billing_control.shipment_transfer_criteria is set to
    "COMPLETED".
    """
    # create an shipment that is ready to bill
    shipment = shipmentFactory()

    shipment_movements = shipment.movements.all()
    shipment_movements.update(status="C")

    shipment.status = "C"
    shipment.billed = False
    shipment.transferred_to_billing = False
    shipment.billing_transfer_date = None
    shipment.save()

    # set the shipment_transfer_criteria on the organization's billing_control
    shipment.organization.billing_control.shipment_transfer_criteria = "COMPLETED"
    shipment.organization.billing_control.save()
    billable_shipments = selectors.get_billable_shipments(
        organization=shipment.organization
    )

    for shipment in billable_shipments:
        assert shipment.status == "C"
        assert not shipment.billed
        assert not shipment.transferred_to_billing
        assert shipment.billing_transfer_date is None


def test_get_billable_shipments_ready_and_completed() -> None:
    """
    Test that get_billable_shipments returns shipments that are completed and not
    billed. When the billing_control.shipment_transfer_criteria is set to
    "READY_AND_COMPLETED".
    """
    shipment = OrderFactory()

    shipment_movements = shipment.movements.all()
    shipment_movements.update(status="C")

    shipment.status = "C"
    shipment.billed = False
    shipment.transferred_to_billing = False
    shipment.billing_transfer_date = None
    shipment.ready_to_bill = True
    shipment.save()

    # set the shipment_transfer_criteria on the organization's billing_control
    shipment.organization.billing_control.shipment_transfer_criteria = (
        "READY_AND_COMPLETED"
    )
    shipment.organization.billing_control.save()
    billable_shipments = selectors.get_billable_orders(
        organization=shipment.organization
    )

    for shipment in billable_orders:
        assert shipment.status == "C"
        assert not shipment.billed
        assert not shipment.transferred_to_billing
        assert shipment.billing_transfer_date is None


def test_get_billable_orders_ready() -> None:
    """
    Test that get_billable_orders returns orders that are completed and not
    billed. When the billing_control.shipment_transfer_criteria is set to
    "READY_AND_COMPLETED".
    """
    # create an shipment that is ready to bill
    shipment = OrderFactory()

    shipment_movements = shipment.movements.all()
    shipment_movements.update(status="C")

    shipment.status = "C"
    shipment.billed = False
    shipment.transferred_to_billing = False
    shipment.billing_transfer_date = None
    shipment.ready_to_bill = True
    shipment.save()

    # set the shipment_transfer_criteria on the organization's billing_control
    shipment.organization.billing_control.shipment_transfer_criteria = "READY_TO_BILL"
    shipment.organization.billing_control.save()
    billable_orders = selectors.get_billable_orders(organization=shipment.organization)

    for shipment in billable_orders:
        assert shipment.status == "C"
        assert not shipment.billed
        assert not shipment.transferred_to_billing
        assert shipment.billing_transfer_date is None


def test_get_billing_queue_information(
    organization: Organization, business_unit: BusinessUnit
) -> None:
    """
    Test that the correct billing queue is returned when using the
    `get_billing_queue_information` selector.
    """

    customer = CustomerFactory()

    shipment = OrderFactory()

    shipment_movements = shipment.movements.all()
    shipment_movements.update(status="C")

    shipment.ready_to_bill = True
    shipment.status = "C"
    shipment.save()

    billing_queue = models.BillingQueue.objects.create(
        organization=organization,
        shipment=shipment,
        customer=customer,
        business_unit=business_unit,
    )

    result = selectors.get_billing_queue_information(shipment=shipment)
    assert result == billing_queue


def test_cannot_delete_billing_history(
    organization: Organization, shipment: Shipment, business_unit: BusinessUnit
) -> None:
    """
    Test that if the organization has remove_billing_history as false that
    the billing history cannot be deleted.
    """
    organization.billing_control.remove_billing_history = False
    organization.billing_control.save()

    shipment.billed = True
    shipment.save()

    billing_history = models.BillingHistory.objects.create(
        organization=organization,
        shipment=shipment,
        customer=shipment.customer,
        business_unit=business_unit,
    )

    with pytest.raises(ValidationError) as excinfo:
        billing_history.delete()

    assert excinfo.value.message_dict["shipment"] == [
        "Your Organization disallows the deletion of billing history. Please try again."
    ]


def test_can_delete_billing_history(
    organization: Organization, shipment: Shipment, business_unit: BusinessUnit
) -> None:
    """
    Test that if the organization has remove_billing_history as true that the billing
    history can be deleted.
    """
    organization.billing_control.remove_billing_history = True
    organization.billing_control.save()

    shipment.billed = True
    shipment.save()

    billing_history = models.BillingHistory.objects.create(
        organization=organization,
        shipment=shipment,
        customer=shipment.customer,
        business_unit=business_unit,
    )

    billing_history.delete()

    assert models.BillingHistory.objects.count() == 0


def test_generate_invoice_number_before_save(
    shipment: Shipment, organization: Organization, business_unit: BusinessUnit
) -> None:
    """
    Test that the invoice number is generated before the save method is called.
    """

    shipment.status = StatusChoices.COMPLETED
    shipment.ready_to_bill = True

    billing_queue = models.BillingQueue.objects.create(
        organization=organization,
        shipment=shipment,
        customer=shipment.customer,
        business_unit=business_unit,
    )

    assert (
        billing_queue.invoice_number
        == f"{shipment.organization.invoice_control.invoice_number_prefix}{billing_queue.shipment.pro_number}".replace(
            "ORD", ""
        )
    )


def test_save_shipment_details_to_billing_history_before_save(
    shipment: Shipment, organization: Organization, business_unit: BusinessUnit
) -> None:
    """
    Test that the shipment details are saved to the billing history before the
    save method is called.
    """
    shipment.billed = True
    shipment.save()

    billing_history = models.BillingHistory.objects.create(
        organization=organization,
        shipment=shipment,
        customer=shipment.customer,
        business_unit=business_unit,
    )

    assert billing_history.pieces == shipment.pieces
    assert billing_history.shipment_type == shipment.shipment_type
    assert billing_history.weight == shipment.weight
    assert billing_history.mileage == shipment.mileage
    assert billing_history.revenue_code == shipment.revenue_code
    assert billing_history.commodity == shipment.commodity
    assert billing_history.bol_number == shipment.bol_number
    assert billing_history.customer == shipment.customer
    assert billing_history.other_charge_total == shipment.other_charge_amount
    assert billing_history.freight_charge_amount == shipment.freight_charge_amount
    assert billing_history.total_amount == shipment.sub_total


def test_transfer_shipment_to_billing_queue(
    organization: Organization, business_unit: BusinessUnit
) -> None:
    """
    Test an shipment is transferred to the billing queue.
    """

    # TODO(Wolfred): Figure out why this test is failing. The transfer item is being created.

    # set the shipment_transfer_criteria on the organization's billing_control
    organization.billing_control.shipment_transfer_criteria = "READY_TO_BILL"
    organization.billing_control.save()

    shipment = OrderFactory(organization=organization, business_unit=business_unit)

    shipment_movements = shipment.movements.all()
    shipment_movements.update(status="C")

    shipment.status = "C"
    shipment.ready_to_bill = True
    shipment.transferred_to_billing = False
    shipment.billing_transfer_date = None
    shipment.save()

    user = UserFactory(organization=organization, business_unit=business_unit)

    services.transfer_to_billing_queue_service(
        user_id=user.id,
        shipment_pros=[shipment.pro_number],
        task_id=str(uuid.uuid4()),
    )

    shipment.refresh_from_db()
    billing_queue = models.BillingQueue.objects.get(shipment=shipment)
    billing_queue.refresh_from_db()

    billing_log_entry = models.BillingLogEntry.objects.get(shipment=shipment)
    billing_log_entry.refresh_from_db()

    assert shipment.transferred_to_billing
    assert shipment.billing_transfer_date is not None
    assert billing_queue.shipment_type == shipment.shipment_type
    assert billing_queue.weight == shipment.weight
    assert billing_queue.pieces == shipment.pieces
    assert billing_queue.revenue_code == shipment.revenue_code
    assert billing_queue.commodity == shipment.commodity
    assert billing_queue.bol_number == shipment.bol_number
    assert billing_queue.customer == shipment.customer
    assert billing_queue.bill_type == "INVOICE"

    # Check that the Billing Log Entry was created
    assert billing_log_entry.shipment == order


def test_bill_orders(
    organization: Organization,
    business_unit: BusinessUnit,
    user: User,
    worker: Worker,
) -> None:
    """
    Test that the orders are billed correctly.
    """

    # set the shipment_transfer_criteria on the organization's billing_control
    organization.billing_control.shipment_transfer_criteria = "READY_TO_BILL"
    organization.billing_control.save()

    # Create an order from the Order Factory
    shipment = OrderFactory(organization=organization, business_unit=business_unit)

    # Update the order movements to be completed
    shipment_movements = shipment.movements.all()
    shipment_movements.update(status="C")

    # Update the order to be ready to bill
    shipment.status = "C"
    shipment.ready_to_bill = True
    shipment.transferred_to_billing = False
    shipment.billing_transfer_date = None
    shipment.save()

    # Create a User from the User Factory
    user = UserFactory(organization=organization, business_unit=business_unit)

    # transfer the order to the billing queue
    services.transfer_to_billing_queue_service(
        user_id=user.id,
        shipment_pros=[shipment.pro_number],
        task_id=str(uuid.uuid4()),
    )

    # Bill all the orders, in the billing queue.
    invoices = models.BillingQueue.objects.all()
    services.bill_orders(user_id=user.id, invoices=invoices, task_id=str(uuid.uuid4()))

    # Query the billing history to make sure it was created.
    billing_history = models.BillingHistory.objects.get(shipment=shipment)
    billing_history.refresh_from_db()

    assert billing_history.shipment == order
    assert billing_history.organization == shipment.organization
    assert billing_history.shipment_type == shipment.shipment_type
    assert billing_history.revenue_code == shipment.revenue_code
    assert billing_history.customer == shipment.customer
    assert billing_history.commodity == shipment.commodity
    assert billing_history.bol_number == shipment.bol_number
    assert (
        billing_history.invoice_number
        == f"{user.organization.invoice_control.invoice_number_prefix}{shipment.pro_number}".replace(
            "ORD", ""
        )
    )

    shipment.refresh_from_db()
    assert shipment.billed is True
    assert shipment.bill_date is not None
    assert mail.outbox[0].subject == f"New invoice from {user.organization.name}"
    assert (
        mail.outbox[0].body
        == f"Please see attached invoice for invoice: {shipment.pro_number}"
    )


def test_single_shipment_billing_service(
    organization: Organization,
    business_unit: BusinessUnit,
    user: User,
    worker: Worker,
) -> None:
    """
    Test an single order can be billed.
    """
    # set the shipment_transfer_criteria on the organization's billing_control
    organization.billing_control.shipment_transfer_criteria = "READY_TO_BILL"
    organization.billing_control.save()

    # Create an order from the Order Factory
    shipment = OrderFactory(organization=organization, business_unit=business_unit)

    # Update the order movements to be completed
    shipment_movements = shipment.movements.all()
    shipment_movements.update(status="C")

    # Update the order to be ready to bill
    shipment.status = "C"
    shipment.ready_to_bill = True
    shipment.transferred_to_billing = False
    shipment.billing_transfer_date = None
    shipment.save()

    # Create a User from the User Factory
    user = UserFactory(organization=organization, business_unit=business_unit)

    # transfer the order to the billing queue
    services.transfer_to_billing_queue_service(
        user_id=user.id,
        shipment_pros=[shipment.pro_number],
        task_id=str(uuid.uuid4()),
    )

    # Bill all the orders, in the billing queue.
    invoice = models.BillingQueue.objects.get(shipment=shipment)
    services.bill_orders(user_id=user.id, invoices=invoice, task_id=str(uuid.uuid4()))

    # Query the billing history to make sure it was created.
    billing_history = models.BillingHistory.objects.get(shipment=shipment)
    billing_history.refresh_from_db()

    assert billing_history.shipment == order
    assert billing_history.organization == shipment.organization
    assert billing_history.shipment_type == shipment.shipment_type
    assert billing_history.revenue_code == shipment.revenue_code
    assert billing_history.customer == shipment.customer
    assert billing_history.commodity == shipment.commodity
    assert (
        billing_history.invoice_number
        == f"{user.organization.invoice_control.invoice_number_prefix}{shipment.pro_number}".replace(
            "ORD", ""
        )
    )
    assert billing_history.bol_number == shipment.bol_number

    shipment.refresh_from_db()
    assert shipment.billed is True
    assert shipment.bill_date is not None
    assert mail.outbox[0].subject == f"New invoice from {user.organization.name}"
    assert (
        mail.outbox[0].body
        == f"Please see attached invoice for invoice: {shipment.pro_number}"
    )


def test_untransfer_single_order(
    api_client: APIClient, organization: Organization, business_unit: BusinessUnit
) -> None:
    shipment = OrderFactory(organization=organization, business_unit=business_unit)

    shipment_movements = shipment.movements.all()
    shipment_movements.update(status="C")

    shipment.status = "C"
    shipment.ready_to_bill = True
    shipment.transferred_to_billing = False
    shipment.billing_transfer_date = None
    shipment.save()
    BillingQueueFactory(shipment=shipment, invoice_number="INV-12345")

    response = api_client.post(
        reverse("untransfer-invoice"), {"invoice_numbers": "INV-12345"}, format="json"
    )

    assert response.status_code == 200
    assert response.data == {"success": "Orders untransferred successfully."}
    shipment.refresh_from_db()
    assert not shipment.transferred_to_billing
    assert shipment.billing_transfer_date is None


def test_untransfer_multiple_orders(
    api_client: APIClient, organization: Organization, business_unit: BusinessUnit
) -> None:
    order1 = OrderFactory(organization=organization, business_unit=business_unit)

    shipment_movements = order1.movements.all()
    shipment_movements.update(status="C")

    order1.status = "C"
    order1.ready_to_bill = True
    order1.transferred_to_billing = False
    order1.billing_transfer_date = None
    order1.save()

    order2 = OrderFactory(organization=organization, business_unit=business_unit)

    shipment_movements = order2.movements.all()
    shipment_movements.update(status="C")

    order2.status = "C"
    order2.ready_to_bill = True
    order2.transferred_to_billing = False
    order2.billing_transfer_date = None
    order2.save()

    BillingQueueFactory(shipment=shipment1, invoice_number="INV-12345")
    BillingQueueFactory(shipment=shipment2, invoice_number="INV-67890")

    response = api_client.post(
        reverse("untransfer-invoice"),
        {"invoice_numbers": ["INV-12345", "INV-67890"]},
        format="json",
    )

    assert response.status_code == 200
    assert response.data == {"success": "Orders untransferred successfully."}
    order1.refresh_from_db()
    order2.refresh_from_db()
    assert not order1.transferred_to_billing
    assert order1.billing_transfer_date is None
    assert not order2.transferred_to_billing
    assert order2.billing_transfer_date is None


def test_validate_invoice_number_does_not_start_with_invoice_prefix(
    organization: Organization, customer: Customer, user: User, worker: Worker
) -> None:
    """
    Test that validates if invoice number is manually entered, it must start with invoice prefix
    from Organization's invoice_control

    Args:
        organization (Organization): Organization object
        customer (Customer): Customer object
        user (User): User object
        worker (Worker): Worker object

    Returns:
        None: This function does return anything.
    """
    shipment = OrderFactory()

    shipment_movements = shipment.movements.all()
    shipment_movements.update(status="C")

    shipment.status = "C"
    shipment.save()

    with pytest.raises(ValidationError) as excinfo:
        models.BillingQueue.objects.create(
            organization=user.organization,
            shipment_type=shipment.shipment_type,
            shipment=shipment,
            revenue_code=shipment.revenue_code,
            customer=customer,
            worker=worker,
            commodity=shipment.commodity,
            bol_number=shipment.bol_number,
            user=user,
            invoice_number="RANDOMINVOICE",
        )

    assert excinfo.value.message_dict["invoice_number"] == [
        "Invoice number must start with invoice prefix from Organization's invoice_control. Please try again."
    ]


def test_validate_invoice_number_does_start_with_invoice_prefix(
    organization: Organization,
    business_unit: BusinessUnit,
    customer: Customer,
    user: User,
    worker: Worker,
) -> None:
    """
    Test that validates if invoice number is manually entered, it must start with invoice prefix
    from Organization's invoice_control

    Args:
        organization (Organization): Organization object
        customer (Customer): Customer object
        user (User): User object
        worker (Worker): Worker object

    Returns:
        None: This function does not return anything.
    """
    shipment = OrderFactory()

    shipment_movements = shipment.movements.all()
    shipment_movements.update(status="C")

    shipment.status = "C"
    shipment.save()

    invoice = models.BillingQueue.objects.create(
        organization=organization,
        business_unit=business_unit,
        shipment_type=shipment.shipment_type,
        shipment=shipment,
        revenue_code=shipment.revenue_code,
        customer=customer,
        worker=worker,
        commodity=shipment.commodity,
        bol_number=shipment.bol_number,
        user=user,
        invoice_number="INV-000001",
    )

    assert invoice.invoice_number == f"INV-{shipment.pro_number}".replace("ORD", "")
