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

from django.db.models.signals import post_save
from django.dispatch import receiver

from integration import models


@receiver(post_save, sender=models.IntegrationVendor)
def create_integration(
    sender: models.IntegrationVendor,
    instance: models.IntegrationVendor,
    created: bool,
    **kwargs: Any,
) -> None:
    """Creates an Integration if a new vendor is added.

    Args:
        sender (IntegrationVendor): IntegrationVendor
        instance (IntegrationVendor): The IntegrationVendor Instance
        created (bool): if the Vendor is created
        **kwargs (Any): Keyword Arguments

    Returns:
        None
    """
    if created:
        models.Integration.objects.create(
            organization=instance.organization, integration_vendor=instance
        )
