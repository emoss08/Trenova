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

from organization import models


@receiver(post_save, sender=models.Depot)
def create_depot_detail(
    sender: models.Depot, instance: models.Depot, created: bool, **kwargs: Any
) -> None:
    """Create Depot Detail Information

    Args:
        sender (Depot): Depot
        instance (Depot): The Depot instance.
        created (bool): if the Depot was created
        **kwargs (Any): Keyword Arguments

    Returns:
        None
    """
    if created:
        models.DepotDetail.objects.create(
            depot=instance, organization=instance.organization
        )


# @receiver(post_save, sender=models.Organization)
# def create_dispatch_control(
#         sender: models.Organization, instance: models.Organization, created: bool, **kwargs: Any
# ) -> None:
#     """Create Dispatch Control Information
#
#     Args:
#         sender (Organization): Organization
#         instance (Organization): The Organization instance.
#         created (bool): if the Organization was created
#         **kwargs (Any): Keyword Arguments
#
#     Returns:
#         None
#     """
#     if created:
#         DispatchControl.objects.create(organization=instance)
