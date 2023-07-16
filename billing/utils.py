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
from typing import Dict, List, LiteralString, Tuple

from django.core.mail import send_mail
from django.utils import timezone

from accounts.models import User
from billing import exceptions, models
from customer.models import Customer, CustomerBillingProfile, CustomerContact
from movements.models import Movement
from order.models import Order
from organization.models import Organization


def delete_invoice(invoice: models.BillingQueue) -> None:
    """Delete a BillingQueue instance.

    Args:
        invoice (models.BillingQueue): An instance of the BillingQueue model.

    Returns:
        None: This function does not return anything.
    """

    invoice.delete()


def set_order_billed(*, order: Order) -> None:
    """Set the billed status of an order to True and set the bill date.

    This function sets the billed status of the passed Order instance to True and sets the bill date to the current
    date and time. It then saves the Order instance.

    Args:
        order (Order): An instance of the Order model.

    Returns:
        None: This function does not return anything.
    """

    order.billed = True
    order.bill_date = timezone.now()
    order.save()


def check_organization_enforces_customer_billing(*, organization: Organization) -> bool:
    """Check if an organization enforces customer billing.

    This function checks if the passed organization enforces customer billing by retrieving the enforce_customer_billing
    field from the organization's BillingControl model.

    Args:
        organization (Organization): An instance of the Organization model.

    Returns:
        bool: A boolean indicating if the passed organization enforces customer billing.
    """

    return bool(organization.billing_control.enforce_customer_billing)


def order_billing_actions(*, invoice: models.BillingQueue, user: User) -> None:
    """Perform billing actions for an order.

    This function performs the necessary billing actions for the passed Order instance. First, it sets the billed status
    of the order to True and sets the bill date. Next, it creates a new BillingHistory instance for the order using
    the create_billing_history function. Then, it deletes the passed BillingQueue instance using the delete_invoice
    function. Finally, it sends an email to the customer with the new invoice attached using the send_billing_email
    function.

    Args:
        invoice (models.BillingQueue): An instance of the BillingQueue model.
        user (User): An instance of the User model who is performing the billing actions.

    Returns:
        None: This function does not return anything.
    """

    set_order_billed(order=invoice.order)
    create_billing_history(invoice=invoice, user=user)
    delete_invoice(invoice=invoice)
    send_billing_email(order=invoice.order, user=user)


def set_order_documents(*, invoice: models.BillingQueue) -> list[str]:
    """Set the document ids for a given order.

    Args:
        invoice (models.BillingQueue): An instance of the BillingQueue model.

    Returns:
        List[str]: A list of the document ids for the passed Order instance.

    This function retrieves the OrderDocument instances for the passed order and creates a list of the
    corresponding document names.
    """

    return [
        document.document_class.name
        for document in invoice.order.order_documentation.all()
        if document.document_class.name
    ]


def create_billing_exception(
    *,
    user: User,
    exception_type: str,
    invoice: models.BillingQueue,
    exception_message: str,
) -> None:
    """Create a new BillingException instance.

    Args:
        user (User): An instance of the User model who is creating the billing exception.
        exception_type (str): A string representing the type of billing exception.
        invoice (models.BillingQueue): An instance of the Order model that the exception pertains to.
        exception_message (str): A string representing the message of the billing exception.

    Returns:
        None: This function does not return anything.
    """

    models.BillingException.objects.create_billing_exception(
        organization=user.organization,
        business_unit=user.organization.business_unit,
        exception_type=exception_type,
        order=invoice.order,
        exception_message=exception_message,
    )


def create_billing_history(*, invoice: models.BillingQueue, user: User) -> None:
    """Create a new BillingHistory instance for an order.

    This function creates a new BillingHistory instance for the passed Order instance. First, it retrieves the corresponding
    Movement instance for the passed order and gets the primary worker if it exists. Next, it creates a new BillingHistory
    instance with the necessary fields using the BillingHistory model. If there is an error creating the BillingHistory
    instance, the function creates a billing exception with the error message and the order information using the
    create_billing_exception function.

    Args:
        invoice (models.BillingQueue): An instance of the Order model.
        user (User): An instance of the User model who is creating the billing history.

    Returns:
        None: This function does not return anything.

    Raises:
        BillingException: If there is an error creating the BillingHistory instance.
    """

    order_movement = Movement.objects.filter(order=invoice.order).first()
    worker = order_movement.primary_worker if order_movement else None

    try:
        models.BillingHistory.objects.create(
            organization=invoice.organization,
            business_unit=invoice.organization.business_unit,
            order=invoice.order,
            worker=worker,
            order_type=invoice.order_type,
            customer=invoice.customer,
            bol_number=invoice.bol_number,
            user=user,
        )
    except exceptions.BillingException as e:
        create_billing_exception(
            user=user,
            exception_type="OTHER",
            invoice=invoice,
            exception_message=f"Error creating billing history: {e}",
        )


def send_billing_email(*, order: Order, user: User) -> None:
    """Email the customer with a new invoice attached.

    This function sends an email to the payable contact of the customer with the new invoice attached. First, the function
    retrieves the payable customer contact for the corresponding customer and organization by filtering the CustomerContact
    model. Next, it retrieves the billing email profile from the organization's EmailControl model to use as the sender email
    address, or if it is not set, it uses the email address of the user who is sending the email. The function then sends an
    email to the customer with the attached invoice using the send_mail function.

    Args:
        order: An instance of the Order model.
        user: An instance of the User model who is sending the email.

    Returns:
        None: This function does not return anything.
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


def set_billing_requirements(*, customer: Customer) -> bool | list[str]:
    """Set the billing requirements for a given customer.

    This function sets the billing requirements for the passed Customer instance by retrieving the corresponding
    billing profile. First, the function checks if the customer has a billing profile with a rule profile. If the
    profile does not exist, it returns False. If the profile exists, the function retrieves the document classes from
    the rule profile and creates a list of the document names. The function then returns the list of billing requirements
    for the customer or False if the profile does not exist.

    Args:
        customer (Customer): A Customer instance.

    Returns:
        bool | List[str]: A list of the billing requirements for the customer or False if the customer does not have a
        billing profile.
    """

    customer_billing_requirements = []

    try:
        if not customer.billing_profile.rule_profile:
            return False

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


def check_billing_requirements(
    *, invoice: models.BillingQueue | Order, user: User
) -> bool | tuple[bool, list[dict[LiteralString, str | list[str]]]]:
    """Check if a BillingQueue instance satisfies the billing requirements of its customer.

    This function checks if the passed BillingQueue instance meets the billing requirements of its corresponding
    customer. First, it sets the billing requirements for the customer using the set_billing_requirements function and
    checks if they exist. If they do not exist, the function creates a billing exception and returns False. Next, the
    function sets the document ids for the corresponding order by calling set_order_documents and checks if the document
    ids match the billing requirements of the customer. If they do not match, the function creates a billing exception
    and returns False. If the document ids match the billing requirements, the function returns True.

    Args:
        invoice (models.BillingQueue): A BillingQueue instance.
        user (User): A User instance.

    Returns:
        bool: True if the BillingQueue instance satisfies the billing requirements of its customer, False otherwise.
    """

    missing_documents = []
    customer_billing_requirements = set_billing_requirements(customer=invoice.customer)
    if customer_billing_requirements is False:
        create_billing_exception(
            user=user,
            exception_type="OTHER",
            invoice=invoice,
            exception_message=f"Customer: {invoice.customer.name} does not have a billing profile",
        )
        return False

    order_document_ids = set_order_documents(invoice=invoice)

    is_match = set(customer_billing_requirements).issubset(  # type: ignore
        set(order_document_ids)
    )
    if not is_match:
        # missing_documents = list(
        #     set(customer_billing_requirements) - set(order_document_ids)  # type: ignore
        # )
        missing_documents.append(
            {
                "invoice_number": invoice.invoice_number,
                "missing_documents": list(
                    set(customer_billing_requirements) - set(order_document_ids)  # type: ignore
                ),
            }
        )
        create_billing_exception(
            user=user,
            exception_type="PAPERWORK",
            invoice=invoice,
            exception_message=f"Missing customer required documents: {missing_documents}",
        )
    return is_match, missing_documents


def transfer_order_details(obj: models.BillingHistory | models.BillingQueue) -> None:
    """Transfer order details from an Order instance to a BillingHistory or BillingQueue instance.

    Args:
        obj: An instance of either BillingHistory or BillingQueue model.

    Returns:
        None.

    Raises:
        Order.DoesNotExist: If the corresponding order does not exist.
    """

    order = Order.objects.select_related(
        "order_type", "revenue_code", "commodity", "customer"
    ).get(pk=obj.order.pk)

    obj.pieces = obj.pieces or order.pieces
    obj.order_type = obj.order_type or order.order_type
    obj.weight = obj.weight or order.weight
    obj.mileage = obj.mileage or order.mileage
    obj.revenue_code = obj.revenue_code or order.revenue_code
    obj.commodity = obj.commodity or order.commodity
    obj.bol_number = obj.bol_number or order.bol_number
    obj.bill_type = obj.bill_type or models.BillingQueue.BillTypeChoices.INVOICE
    obj.bill_date = obj.bill_date or timezone.now().date()
    obj.consignee_ref_number = obj.consignee_ref_number or order.consignee_ref_number
    if obj.commodity and not obj.commodity_descr:
        obj.commodity_descr = obj.commodity.description

    obj.customer = order.customer
    obj.other_charge_total = order.other_charge_amount
    obj.freight_charge_amount = order.freight_charge_amount
    obj.total_amount = order.sub_total
