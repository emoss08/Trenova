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

from django.core.exceptions import ValidationError
from django.utils.translation import gettext_lazy as _

from billing import models, services, utils
from billing.selectors import get_billing_queue_information


def prevent_delete_on_rate_con_doc_class(
    instance: models.DocumentClassification,
    **kwargs: Any,
) -> None:
    """
    Prevents the deletion of the Document Classification with name "CON"

    Args:
        instance (models.DocumentClassification): Document Classification instance
        **kwargs (Any): Any additional arguments

    Returns:
        None: This function does not return anything
    """
    if instance.name == "CON":
        raise ValidationError(
            {
                "name": _(
                    "Document classification with this name cannot be deleted. Please try again."
                ),
            },
            code="invalid",
        )


def check_billing_history(
    instance: models.BillingHistory,
    **kwargs: Any,
) -> None:
    """
    Prevents the deletion of the Billing History if the organization has the remove_billing_history

    Args:
        instance (models.BillingHistory): Billing History instance
        **kwargs (Any): Any additional arguments

    Returns:
        None: This function does not return anything

    Raises:
        ValidationError: If the organization has the remove_billing_history set to False
    """
    if instance.organization.billing_control.remove_billing_history is False:
        raise ValidationError(
            {
                "organization": _(
                    "Billing history cannot be deleted. Please try again."
                ),
            },
            code="invalid",
        )


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
    if billing_queue := get_billing_queue_information(order=instance.order):
        instance.invoice_number = billing_queue.invoice_number


def transfer_order_details_to_billing_history(
    instance: models.BillingHistory, **kwargs: Any
) -> None:
    """Transfers the order details to the billing history

    Args:
        instance (models.BillingHistory): Billing History instance
        **kwargs (Any): Any additional arguments

    Returns:
        None: This function does not return anything
    """
    utils.transfer_order_details(obj=instance)


def generate_invoice_number_on_billing_queue(
    instance: models.BillingQueue, **kwargs: Any
) -> None:
    is_credit_memo = instance.bill_type == models.BillingQueue.BillTypeChoices.CREDIT

    if not instance.invoice_number:
        services.generate_invoice_number(
            instance=instance, is_credit_memo=is_credit_memo
        )


def transfer_order_details_to_billing_queue(
    instance: models.BillingQueue, **kwargs: Any
) -> None:
    """Transfers the order details to the billing queue

    Args:
        instance (models.BillingQueue): Billing Queue instance
        **kwargs (Any): Any additional arguments

    Returns:
        None: This function does not return anything
    """
    utils.transfer_order_details(obj=instance)
