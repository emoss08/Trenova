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
from typing import Iterator

from django.core.mail import send_mail
from django.db import IntegrityError, transaction
from django.http import HttpRequest
from django.utils import timezone

from accounts.models import User
from billing import models
from billing.selectors import get_billable_orders
from customer.models import Customer, CustomerBillingProfile, CustomerContact
from order.models import Order
from silk.profiling.profiler import silk_profile


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
        self.order_document_ids: list[str] = []
        self.customer_billing_requirements: list[str] = []
        self.billing_queue: Iterable[models.BillingQueue] = []

    def create_billing_exception(
            self, *, exception_type: str, order: Order | None, exception_message: str
    ) -> None:
        """Create a billing Exception

        Args:
            exception_type (str): The type of exception
            order (Order | None): The order that caused the exception
            exception_message (str): Description of the Exception

        Returns:
            None: None
        """
        models.BillingException.objects.create(
            organization=self.request.user.organization,
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
            self.customer_billing_requirements.extend(
                [
                    doc.name
                    for doc in customer.billing_profile.rule_profile.document_class.all()
                    if doc.name
                ]
            )
        except CustomerBillingProfile.DoesNotExist:
            self.create_billing_exception(
                exception_type="OTHER",
                order=order,
                exception_message=f"Customer: {customer.name} does not have a billing profile",
            )
        return

    def set_order_document_ids(self, *, order: Order) -> None:
        """Set the order document ids for the order

        Args:
            order (Order): The order to set the order document ids for

        Returns:
            None: None
        """

        self.order_document_ids = [
            document.document_class.name
            for document in order.order_documentation.all()
            if document.document_class.name
        ]

    def get_billing_queue(self) -> Iterable[models.BillingQueue]:
        """Get the billing queue for the organization

        Returns:
            QuerySet: The billing queue queryset
        """
        self.billing_queue = models.BillingQueue.objects.filter(
            organization=self.request.user.organization
        )
        return self.billing_queue

    def check_billing_requirements(self, *, order: Order) -> bool:
        """Check if the billing requirements are met

        Returns:
            bool: True if the billing requirements are met, False otherwise
        """
        is_match = set(self.customer_billing_requirements).issubset(
            set(self.order_document_ids)
        )
        if not is_match:
            missing_documents = list(
                set(self.customer_billing_requirements) - set(self.order_document_ids)
            )
            self.create_billing_exception(
                exception_type="PAPERWORK",
                order=order,
                exception_message=f"Missing customer required documents: {missing_documents}",
            )
        return is_match

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
        """Create billing history for the given order.

        Args:
            order (Order): The order to create the billing history for.

        Returns:
            None

        Raises:
            BillingException: If there is an error creating the billing history.

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
                user=self.request.user,
            )
        except BillingException as e:
            self.create_billing_exception(
                exception_type="OTHER",
                order=order,
                exception_message=f"Error creating billing history: {e}",
            )

    def send_billing_email(self, *, order: Order) -> None:
        """Sends billing email to the customer contact.

        This function is used to send an email with the billing invoice to the customer contact,
        which is either set as the payable contact in the customer organization or the default
        contact email of the organization.

        Args:
            order (Order): The order for which the billing email is sent.

        Returns:
            None: This function does not return any value.
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

    @silk_profile(name="Bill Orders")  # type: ignore
    def bill_orders(self) -> None:
        """Bill a list of orders to their respective customers

        This function bills a list of orders to their respective customers.
        For each order in the billing queue, the function first checks if
        the billing control is enabled. If it is, it sets the billing requirements
        for the customer and order, sets the order document IDs, and checks if the
        billing requirements are met. If the requirements are met, it sets the order
        as billed, sends a billing email to the customer, and creates a billing history
        record. If the requirements are not met, a billing exception is created. If the
        billing control is not enabled, the order is simply set as billed. Finally, the
        invoice is deleted from the billing queue.

        Returns:
            None: None
        """
        for invoice in self.get_billing_queue():
            if self._check_billing_control():
                self.set_billing_requirements(
                    customer=invoice.order.customer, order=invoice.order
                )
                self.set_order_document_ids(order=invoice.order)
                if self.check_billing_requirements(order=invoice.order):
                    with transaction.atomic():
                        self.set_order_billed(order=invoice.order)
                        self.create_billing_history(order=invoice.order)
                        self.delete_billing_queue(billing_queue=invoice)
                    self.send_billing_email(order=invoice.order)
            else:
                with transaction.atomic():
                    self.set_order_billed(order=invoice.order)
                    self.create_billing_history(order=invoice.order)
                    self.delete_billing_queue(billing_queue=invoice)
                self.send_billing_email(order=invoice.order)

    def bill_order(self, *, order: Order) -> None:
        """Bill a single order by performing multiple operations.

        This function checks if the billing control is valid, sets the billing requirements,
        sets the order document IDs, checks if the billing requirements are met, sets the order
        as billed, and sends a billing email if the requirements are met. If the billing control is
        not valid, or if the billing requirements are not met, an exception is created.

        Args:
            order (Order): The order to bill

        Returns:
            None: None

        Raises:
            None
        """
        if self._check_billing_control():
            self.set_billing_requirements(customer=order.customer, order=order)
            self.set_order_document_ids(order=order)
            if self.check_billing_requirements(order=order):
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

    @silk_profile(name="Transfer To Billing Queue")  # type: ignore
    def transfer_to_billing_queue(self) -> None:
        """
        Transfer billable orders to the billing queue for the current organization.

        This function retrieves the billable orders for the current organization and transfers them
        to the billing queue. The transferred orders are marked as `transferred_to_billing` and their
        `billing_transfer_date` is set to the current date and time.

        Returns:
            None: This function returns `None`.
        """

        billable_orders: Iterator[Order] | None = get_billable_orders(
            organization=self.request.user.organization
        )

        if not billable_orders:
            # TODO: Decide if we want to log this, or give this information to the user.
            return

        order_ids = [order.id for order in billable_orders]
        now = timezone.now()

        Order.objects.filter(id__in=order_ids).update(
            transferred_to_billing=True, billing_transfer_date=now
        )

        try:

            for order in order_ids:
                models.BillingQueue.objects.create(
                    organization=self.request.user.organization, order_id=order
            )

            bill_transfer_logs = [
                models.BillingTransferLog(
                    organization=self.request.user.organization,
                    order_id=order_id,
                    transferred_at=now,
                    transferred_by=self.request.user,
                )
                for order_id in order_ids
            ]
            models.BillingTransferLog.objects.bulk_create(bill_transfer_logs)

        except IntegrityError as int_err:
            self.create_billing_exception(
                exception_type="OTHER",
                order=None,
                exception_message=f"Error transferring orders to billing queue: {int_err}",
            )
        except BillingException as bill_err:
            self.create_billing_exception(
                exception_type="OTHER",
                order=None,
                exception_message=f"Error transferring orders to billing queue: {bill_err}",
            )
