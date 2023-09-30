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

from typing import Any

from billing import models, services, utils
from billing.selectors import get_billing_queue_information


def save_invoice_number_on_billing_history(
    instance: models.BillingHistory, **kwargs: Any
) -> None:
    """Saves the invoice number on the billing history

    Args:
        instance (models.BillingHistory): Billing History instance
        **kwargs (Any): Any additional arguments

    Returns:
        None: This function does not return anything
    """
    if billing_queue := get_billing_queue_information(shipment=instance.shipment):
        instance.invoice_number = billing_queue.invoice_number


def transfer_shipments_details_to_billing_history(
    instance: models.BillingHistory, **kwargs: Any
) -> None:
    """Transfers the shipment details to the billing history

    Args:
        instance (models.BillingHistory): Billing History instance
        **kwargs (Any): Any additional arguments

    Returns:
        None: This function does not return anything
    """
    utils.transfer_shipments_details(obj=instance)


def generate_invoice_number_on_billing_queue(
    instance: models.BillingQueue, **kwargs: Any
) -> None:
    if not instance.invoice_number:
        is_credit_memo = (
            instance.bill_type == models.BillingQueue.BillTypeChoices.CREDIT
        )

        services.generate_invoice_number(
            instance=instance, is_credit_memo=is_credit_memo
        )


def transfer_shipments_details_to_billing_queue(
    instance: models.BillingQueue, **kwargs: Any
) -> None:
    """Transfers the shipment details to the billing queue

    Args:
        instance (models.BillingQueue): Billing Queue instance
        **kwargs (Any): Any additional arguments

    Returns:
        None: This function does not return anything
    """
    utils.transfer_shipments_details(obj=instance)
