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
from collections.abc import Iterable

from django.core.mail import send_mail
from django.http import HttpRequest
from django.utils import timezone
from notifications.signals import notify

from accounts.models import User
from billing import models
from customer.models import Customer, CustomerBillingProfile, CustomerContact
from movements.models import Movement
from order.models import Order


class AuthenticatedHTTPRequest(HttpRequest):
    """
    Authenticated HTTP Request
    """

    user: User


class BillingException(Exception):
    """
    Base Billing Exception
    """


def create_billing_exception(
    *, user: User, exception_type: str, order: Order | None, exception_message: str
) -> None:
    """Create a billing Exception

    Args:
        user (User): The user that caused the exception
        exception_type (str): The type of exception
        order (Order | None): The order that caused the exception
        exception_message (str): Description of the Exception

    Returns:
        None: None
    """
    models.BillingException.objects.create(
        organization=user.organization,
        exception_type=exception_type,
        order=order,
        exception_message=exception_message,
    )


def check_billing_control(*, user: User) -> bool:
    """Check billing control for the organization.

    Check if the organization `enforce_customer_billing` is set to True/

    Returns:
        bool: True if billing control is set to True, False otherwise
    """
    return bool(user.organization.billing_control.enforce_customer_billing)


def set_billing_requirements(*, customer: Customer) -> list[str] | bool:
    """Set the billing requirements for the customer

    Args:
        customer (Customer): The customer to set the billing requirements for

    Returns:
        None: None
    """

    customer_billing_requirements = []

    try:
        customer_billing_requirements.extend(
            [
                doc.name
                for doc in customer.billing_profile.rule_profile.document_class.all()
                if doc.name
            ]
        )
    except CustomerBillingProfile.DoesNotExist:
        return False

    return customer_billing_requirements


def set_order_documents(*, order: Order) -> list[str]:
    """Set the order document ids for the order

    Args:
        order (Order): The order to set the order document ids for

    Returns:
        None: None
    """

    return [
        document.document_class.name
        for document in order.order_documentation.all()
        if document.document_class.name
    ]


def get_billing_queue(*, user: User, task_id: str) -> Iterable[models.BillingQueue]:
    """Get the billing queue for the organization

    Returns:
        QuerySet: The billing queue queryset
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


def check_billing_requirements(*, order: Order, user: User) -> bool:
    """Check if the billing requirements are met

    Returns:
        bool: True if the billing requirements are met, False otherwise
    """

    customer_billing_requirements = set_billing_requirements(customer=order.customer)
    if customer_billing_requirements is False:
        create_billing_exception(
            user=user,
            exception_type="OTHER",
            order=order,
            exception_message=f"Customer: {order.customer.name} does not have a billing profile",
        )
        return False

    order_document_ids = set_order_documents(order=order)

    is_match = set(customer_billing_requirements).issubset(  # type: ignore
        set(order_document_ids)
    )
    if not is_match:
        missing_documents = list(
            set(customer_billing_requirements) - set(order_document_ids)  # type: ignore
        )
        create_billing_exception(
            user=user,
            exception_type="PAPERWORK",
            order=order,
            exception_message=f"Missing customer required documents: {missing_documents}",
        )
    return is_match


def set_order_billed(*, order: Order) -> None:
    """Set the order billed

    Args:
        order (Order): The order to set billed

    Returns:
        None: None
    """
    order.billed = True
    order.bill_date = timezone.now()
    order.save()


def delete_billing_queue(*, billing_queue: models.BillingQueue) -> None:
    """Delete the billing queue

    Args:
        billing_queue (models.BillingQueue): The billing queue to delete

    Returns:
        None: None
    """
    billing_queue.delete()


def create_billing_history(*, order: Order, user: User) -> None:
    """Create billing history for the given order.

    Args:
        order (Order): The order to create the billing history for.
        user (User): The user that created the billing history.

    Returns:
        None

    Raises:
        BillingException: If there is an error creating the billing history.

    """

    order_movement = Movement.objects.filter(order=order).first()
    worker = order_movement.primary_worker if order_movement else None

    try:
        models.BillingHistory.objects.create(
            organization=order.organization,
            order=order,
            worker=worker,
            order_type=order.order_type,
            customer=order.customer,
            bol_number=order.bol_number,
            user=user,
        )
    except BillingException as e:
        create_billing_exception(
            user=user,
            exception_type="OTHER",
            order=order,
            exception_message=f"Error creating billing history: {e}",
        )


def send_billing_email(*, order: Order, user: User) -> None:
    """Sends billing email to the customer contact.

    This function is used to send an email with the billing invoice to the customer contact,
    which is either set as the payable contact in the customer organization or the default
    contact email of the organization.

    Args:
        user (User): The user that caused the exception
        order (Order): The order for which the billing email is sent.

    Returns:
        None: This function does not return any value.
    """
    customer_contact = CustomerContact.objects.filter(
        customer=order.customer,
        organization=user.organization,
        is_payable_contact=True,
    ).first()

    billing_profile = user.organization.email_control.billing_email_profile

    send_mail(
        f"New invoice from {user.organization.name}",
        f"Please see attached invoice for invoice: {order.pro_number}",
        f"{billing_profile.email if billing_profile else user.email}",
        [customer_contact.email if customer_contact else user.email],
        fail_silently=False,
    )
