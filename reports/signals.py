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

import contextlib
import logging
from typing import Any

from django_celery_beat.models import PeriodicTask

from reports import models, services

logger = logging.getLogger(__name__)


def update_scheduled_task(
    sender: models.ScheduledReport, instance: models.ScheduledReport, **kwargs: Any
) -> None:
    """
    Update the scheduled task when the day of the week is changed.

    Args:
        sender (models.ScheduledReport): Sender of the signal.
        instance (models.ScheduledReport): The instance of the sender.
        **kwargs (Any): Additional keyword arguments.

    Returns:
        None: This function does not return anything.
    """

    services.create_scheduled_task(instance=instance)


def delete_scheduled_report_periodic_task(
    sender: models.ScheduledReport, instance: models.ScheduledReport, **kwargs: Any
) -> None:
    """
    Delete the scheduled task when the scheduled report is deleted.

    Args:
        sender (models.ScheduledReport): Sender of the signal.
        instance (models.ScheduledReport): The instance of the sender.
        **kwargs (Any): Additional keyword arguments.

    Returns:
        None: This function does not return anything.
    """
    with contextlib.suppress(PeriodicTask.DoesNotExist):
        logger.log(
            level=logging.INFO,
            msg=f"Deleting scheduled task for scheduled report {instance.user_id}-{instance.pk}",
        )
        periodic_task = PeriodicTask.objects.get(
            name=f"Send scheduled report {instance.user_id}-{instance.pk}"
        )
        periodic_task.delete()
