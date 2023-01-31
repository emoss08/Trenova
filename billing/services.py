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

import uuid
from collections.abc import Iterable

from django.core.mail import send_mail
from django.http import HttpRequest
from django.utils import timezone

from accounts.models import User
from billing import models
from customer.models import Customer, CustomerBillingProfile, CustomerContact
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

    pass


class BillingService:
    """
    Class to handle billing to customers
    """

    def __init__(self, *, request: AuthenticatedHTTPRequest):
        self.request = request
        self.order_document_ids: list[uuid.UUID] = []
        self.billing_requirements: list = []
        self.billing_queue: Iterable[models.BillingQueue] = []
        self.bill_orders()

    @staticmethod
    def create_billing_exception(
        *, exception_type: str, order: Order, exception_message: str
    ) -> None:
        """Create a billing Exception

        Args:
            exception_type (str): The type of exception
            order (Order): The order that caused the exception
            exception_message (str): Description of the Exception

        Returns:
            None: None
        """
        models.BillingException.objects.create(
            organization=order.organization,
            exception_type=exception_type,
            order=order,
            exception_message=exception_message,
        )

    def _check_billing_control(self) -> bool:
        """Check billing control for the organization.

        Check if the organization `enforce_customer_billing` is set to True/

        Returns:
            bool: True if billing control is set to True, False otherwise
        """
        return bool(
            self.request.user.organization.billing_control.enforce_customer_billing
        )

    def set_billing_requirements(self, *, customer: Customer, order: Order) -> None:
        """Set the billing requirements for the customer

        Args:
            customer (Customer): The customer to set the billing requirements for
            order (Order): The order to set the billing requirements for

        Returns:
            None: None
        """
        try:
            customer_billing_profile = CustomerBillingProfile.objects.get(
                customer=customer, organization=customer.organization
            )
        except CustomerBillingProfile.DoesNotExist:
            msg = f"""
            Customer Billing Profile Not found for Customer {customer.name},
            or Customer does not have a valid email address. No need to worry, we've
            sent the invoices to {self.request.user.email}. Please update the
            Customer Billing Profile for {customer.name} and try again.
            """
            self.create_billing_exception(
                exception_type="OTHER",
                order=order,
                exception_message=msg,
            )
            return

        self.billing_requirements.extend(
            customer_billing_profile.values_list("document_class", flat=True)
        )

    def set_order_document_ids(self, *, order: Order) -> None:
        """Set the order document ids for the order

        Args:
            order (Order): The order to set the order document ids for

        Returns:
            None: None
        """
        self.order_document_ids.extend(
            document.document_class.id for document in order.order_documentation.all()
        )

    def get_billing_queue(self) -> Iterable[models.BillingQueue]:
        """Get the billing queue for the organization

        Returns:
            QuerySet: The billing queue queryset
        """
        self.billing_queue = models.BillingQueue.objects.filter(
            organization=self.request.user.organization
        )
        return self.billing_queue

    def check_billing_requirements(self) -> bool:
        """Check if the billing requirements are met

        Returns:
            bool: True if the billing requirements are met, False otherwise
        """
        return set(self.billing_requirements).issubset(set(self.order_document_ids))

    @staticmethod
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

    @staticmethod
    def delete_billing_queue(*, billing_queue: models.BillingQueue) -> None:
        """Delete the billing queue

        Args:
            billing_queue (models.BillingQueue): The billing queue to delete

        Returns:
            None: None
        """
        billing_queue.delete()

    def create_billing_history(self, *, order: Order) -> None:
        """Create the billing history

        Args:
            order (Order): The order to create the billing history for

        Returns:
            None: None
        """

        order_movement = order.movements.first()
        worker = order_movement.primary_worker if order_movement else None

        try:
            models.BillingHistory.objects.create(
                organization=order.organization,
                order=order,
                worker=worker,
                order_type=order.order_type,
                customer=order.customer,
                bol_number=order.bol_number,
            )
        except BillingException as e:
            self.create_billing_exception(
                exception_type="OTHER",
                order=order,
                exception_message=f"Error creating billing history: {e}",
            )

    def send_billing_email(self, *, order: Order) -> None:
        """Send the billing email

        Args:
            order (Order): The order to send the billing email for

        Returns:
            None: None
        """
        customer_contact = CustomerContact.objects.filter(
            customer=order.customer,
            organization=self.request.user.organization,
            is_payable_contact=True,
        ).first()
        billing_profile = (
            self.request.user.organization.email_control.billing_email_profile
        )
        send_mail(
            f"New invoice from {self.request.user.organization.name}",
            f"Please see attached invoice for invoice: {order.pro_number}",
            f"{billing_profile.email if billing_profile else self.request.user.email}",
            [customer_contact.email if customer_contact else self.request.user.email],
            fail_silently=False,
        )

    def bill_orders(self) -> None:
        """Bill orders to customers

        Returns:
            None: None
        """
        for invoice in self.get_billing_queue():
            if self._check_billing_control():
                self.set_billing_requirements(
                    customer=invoice.order.customer, order=invoice.order
                )
                self.set_order_document_ids(order=invoice.order)
                if self.check_billing_requirements():
                    self.set_order_billed(order=invoice.order)
                    self.send_billing_email(order=invoice.order)
                    self.create_billing_history(order=invoice.order)
                else:
                    self.create_billing_exception(
                        exception_type="PAPERWORK",
                        order=invoice.order,
                        exception_message="Billing requirement not met",
                    )
            else:
                self.set_order_billed(order=invoice.order)

            self.delete_billing_queue(billing_queue=invoice)

    def bill_order(self, *, order: Order) -> None:
        """Bill a single order

        Args:
            order (Order): The order to bill

        Returns:
            None: None
        """
        if self._check_billing_control():
            self.set_billing_requirements(customer=order.customer, order=order)
            self.set_order_document_ids(order=order)
            if self.check_billing_requirements():
                self.set_order_billed(order=order)
                self.send_billing_email(order=order)
            else:
                self.create_billing_exception(
                    exception_type="PAPERWORK",
                    order=order,
                    exception_message="Billing requirement not met",
                )
        else:
            self.set_order_billed(order=order)
