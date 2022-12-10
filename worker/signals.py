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

from worker.services import generation

from .models import Worker, WorkerProfile


@receiver(pre_save, sender=Worker)
def generate_worker_code(sender: Worker, instance: Worker, **kwargs: Any) -> None:
    """Generate Worker Code

    Generate a worker code when a new worker is added.

    Args:
        sender (Worker): Worker
        instance (Worker): The worker instance.
        **kwargs (Any): Keyword arguments.

    Returns:
        None
    """
    if not instance.code:
        instance.code = generation.WorkerGenerationService.worker_code(instance)


# @receiver(post_save, sender=Worker)
# def create_worker_profile(
#     sender: Worker, instance: Worker, created: bool, **kwargs: Any
# ) -> None:
#     """Create Worker Profile
#
#     Create a worker profile when a new worker is added.
#
#     Args:
#         sender (Worker): Worker
#         instance (Worker): The worker instance.
#         created (bool): If the worker was created.
#         **kwargs (Any): Keyword arguments.
#
#         Returns:
#             None:
#     """
#     if created:
#         WorkerProfile.objects.create(
#             worker=instance, organization=instance.organization
#         )
