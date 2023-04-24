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

from worker import models
from worker.services.generation import WorkerGenerationService


def create_worker_code(
    sender: models.Worker, instance: models.Worker, **kwargs: Any
) -> None:
    """Generate a worker code for a Worker model instance if it doesn't exist.

    This function is called as a signal when a Worker model instance is saved.
    If the worker instance does not have a code, it generates a worker code
    using the WorkerGenerationService.

    Args:
        sender (models.Worker): The class of the sending instance.
        instance (models.Worker): The instance of the Worker model being saved.
        **kwargs: Additional keyword arguments.
    """
    if not instance.code:
        WorkerGenerationService.generate_worker_code(instance=instance)


def create_worker_profile(
    sender: models.Worker, instance: models.Worker, created: bool, **kwargs: Any
) -> None:
    """Create a WorkerProfile model instance for a new Worker model instance.

    This function is called as a signal when a Worker model instance is saved.
    If a new Worker instance is created and it does not have an associated
    WorkerProfile, it creates a WorkerProfile model instance with the worker
    and organization references.

    Args:
        sender (models.Worker): The class of the sending instance.
        instance (models.Worker): The instance of the Worker model being saved.
        created (bool): True if a new record was created, False otherwise.
        **kwargs: Additional keyword arguments.
    """
    if (
        created
        and not models.WorkerProfile.objects.filter(pk__exact=instance.pk).exists()
    ):
        models.WorkerProfile.objects.create(
            worker=instance, organization=instance.organization
        )
