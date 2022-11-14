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

from customer.services import generation

from .models import Customer, CustomerBillingProfile


@receiver(pre_save, sender=Customer)
def generate_customer_code(sender: Customer, instance: Customer, **kwargs: Any) -> None:
    """Generate Customer Code

    Generate a customer code when a new worker is added.

    Args:
        sender (Customer): Customer
        instance (Customer): The customer instance.
        **kwargs (Any): Keyword arguments.

    Returns:
        None
    """
    if not instance.code:
        instance.code = generation.CustomerGenerationService.customer_code(instance)


@receiver(post_save, sender=Customer)
def create_customer_billing_profile(
    sender: CustomerBillingProfile,
    instance: CustomerBillingProfile,
    created: bool,
    **kwargs: Any
) -> None:
    """Create Customer Billing Profile

    Args:
        sender (CustomerBillingProfile): Customer Billing Profile.
        instance (EquipmentType): The CustomerBillingProfile instance.
        created (bool): if the Customer Billing Profile was created
        **kwargs (Any): Keyword Arguments

    Returns:
        None
    """
    if created:
        CustomerBillingProfile.objects.create(
            customer=instance, organization=instance.organization
        )
