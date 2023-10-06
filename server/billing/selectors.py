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

from django.db.models import Q, QuerySet
from notifications.signals import notify

from billing import models
from utils.models import StatusChoices

if TYPE_CHECKING:
    from accounts.models import User
    from organization.models import Organization
    from shipment.models import Shipment
    from utils.types import ModelUUID


def get_billable_shipments(
    *, organization: "Organization", shipment_pros: list[str] | None = None
) -> QuerySet["Shipment"] | None:
    """Retrieve billable shipments for a given organization based on specified criteria.

    Args:
        organization (Organization): The organization for which to retrieve billable shipments.
        shipment_pros (List[str] | None, optional): A list of shipment PRO numbers to filter by, if specified.
            Defaults to None.

    Returns:
        QuerySet[Shipment] | None: A queryset of billable shipments, or None if no billable shipments are found.
    """

    # Map BillingControl.shipmentTransferCriteriaChoices to the corresponding query
    criteria_to_query = {
        models.BillingControl.ShipmentTransferCriteriaChoices.READY_AND_COMPLETED: Q(
            status=StatusChoices.COMPLETED
        )
        & Q(ready_to_bill=True),
        models.BillingControl.ShipmentTransferCriteriaChoices.COMPLETED: Q(
            status=StatusChoices.COMPLETED
        ),
        models.BillingControl.ShipmentTransferCriteriaChoices.READY_TO_BILL: Q(
            ready_to_bill=True
        ),
    }

    query: Q = (
        Q(billed=False)
        & Q(transferred_to_billing=False)
        & Q(billing_transfer_date__isnull=True)
    )
    shipment_criteria_query: Q | None = criteria_to_query.get(
        organization.billing_control.shipment_transfer_criteria
    )

    if shipment_criteria_query is not None:
        query &= shipment_criteria_query

    if shipment_pros:
        query &= Q(pro_number__in=shipment_pros)

    shipments = organization.shipments.filter(query)

    return shipments if shipments.exists() else None


def get_billing_queue_information(
    *, shipment: "Shipment"
) -> models.BillingQueue | None:
    """Retrieve the most recent billing queue information for a given shipment.

    Args:
        shipment (Shipment): The shipment for which to retrieve billing queue information.

    Returns:
        models.BillingQueue | None: The most recent BillingQueue instance for the given order,
            or None if no billing queue information is found.
    """
    return models.BillingQueue.objects.filter(shipment=shipment).last()


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
            verb="Shipment Billing Exception",
            description=f"No shipments in the billing queue for task: {task_id}",
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


def get_billing_history_by_customer_id(
    *, customer_id: uuid.UUID
) -> QuerySet[models.BillingHistory]:
    """Retrieves a queryset of BillingHistory objects by their customer id.

    Args:
        customer_id (uuid.UUID):

    Returns:
        QuerySet[models.BillingHistory]: A queryset of BillingHistory objects.
    """
    return models.BillingHistory.objects.filter(customer_id__exact=customer_id)


def get_unpaid_invoices() -> QuerySet[models.BillingHistory]:
    """Get all the unpaid invoices from the Billing History.

    This function returns a QuerySet containing all the Billing History model objects that have not received payment yet.
    The filter applies on 'payment_received' attribute of objects, if it's False then it's deemed as unpaid invoice.

    Returns:
        QuerySet[models.BillingHistory]: A QuerySet containing all the unpaid invoices.
    """
    return models.BillingHistory.objects.filter(payment_received=False)


def get_paid_invoices() -> QuerySet[models.InvoicePaymentDetail]:
    """Get all the paid invoices from the Invoice Payment Detail.

    This function returns a QuerySet containing all the Invoice Payment Detail model objects where payment has been received.
    An invoice is classified as paid if the 'payment_date' and 'payment_amount' attributes are not null which essentially mean a payment has been made.

    Returns:
        QuerySet[models.InvoicePaymentDetail]: A QuerySet containing all the paid invoices.
    """
    return models.InvoicePaymentDetail.objects.filter(
        payment_date__isnull=False, payment_amount__isnull=False
    )


def get_shipment_bill_hist(*, shipment: "Shipment") -> models.BillingHistory:
    """Retrieves a queryset of BillingHistory objects by their shipment.

    Args:
        shipment (Shipment): The shipment for which to retrieve billing history.

    Returns:
        QuerySet[models.BillingHistory]: A queryset of BillingHistory objects.
    """
    return models.BillingHistory.objects.get(shipment=shipment)


def get_invoice_payment_detail(
    *, invoice: models.BillingHistory
) -> models.InvoicePaymentDetail:
    """Retrieves a queryset of BillingHistory objects by their shipment.

    Args:
        invoice (models.BillingHistory): The shipment for which to retrieve billing history.

    Returns:
        QuerySet[models.InvoicePaymentDetail]: A queryset of BillingHistory objects.
    """

    return models.InvoicePaymentDetail.objects.get(invoice=invoice)
