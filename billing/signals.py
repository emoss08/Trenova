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

from typing import Any

from django.db.models.signals import post_save, pre_save
from django.dispatch import receiver

from billing import models
from billing.services.transfer_order_details import TransferOrderDetails
from organization.models import Organization


@receiver(post_save, sender=Organization)
def create_billing_control(
    sender: Organization,
    instance: Organization,
    created: bool,
    **kwargs: Any,
) -> None:
    """Create Billing Control information.

    Args:
        sender (Organization): Organization
        instance (Organization): The Organization instance
        created (bool): if the Organization was created
        **kwargs (Any): Keyword Arguments

    Returns:
        None
    """
    if created:
        models.BillingControl.objects.create(organization=instance)

@receiver(pre_save, sender=models.BillingQueue)
def save_billing_queue_order_details(sender: models.BillingQueue, instance: models.BillingQueue, **kwargs: Any) -> None:
    """Save order details

    Call the transfer_order_details service, select order details and save them to billingqueue objects.

    Args:
        sender (models.BillingQueue): BillingQueue
        instance (models.BillingQueue): The BillingQueue instance.
        **kwargs (Any): Keyword Arguments

    Returns:
        None
    """
    TransferOrderDetails(model=instance)

@receiver(pre_save, sender=models.BillingHistory)
def save_billing_history_order_details(sender: models.BillingHistory, instance: models.BillingHistory, **kwargs: Any) -> None:
    """Save order details

    Call the transfer_order_details service, select order details and save them to billingqueue objects.

    Args:
        sender (models.BillingHistory): BillingHistory
        instance (models.BillingHistory): The BillingHistory instance.
        **kwargs (Any): Keyword Arguments

    Returns:
        None
    """
    TransferOrderDetails(model=instance)
