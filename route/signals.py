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

from organization.models import Organization
from .models import RouteControl


@receiver(post_save, sender=Organization)
def create_route_control(
    sender: Organization, instance: Organization, created: bool, **kwargs: Any
) -> None:
    """Create Route Control Information

    Args:
        sender (Organization): Organization
        instance (Organization): The Organization instance.
        created (bool): if the Organization was created
        **kwargs (Any): Keyword Arguments

    Returns:
        None
    """
    if created:
        RouteControl.objects.create(organization=instance)
