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
from typing import TYPE_CHECKING

from celery import shared_task
from celery_singleton import Singleton

from core.exceptions import ServiceException
from dispatch import selectors

if TYPE_CHECKING:
    from celery.app.task import Task


@shared_task(bind=True, max_retries=3, default_retry_delay=60 * 60 * 24, base=Singleton)
def send_expired_rates_notification(self: "Task") -> None:
    """Send expired rates notification

    Args:
        self (celery.app.task.Task): The task object

    Returns:
        None: This function does not return anything.
    """

    expired_rates = selectors.get_expired_rates()
    if not expired_rates:
        print("No expired rates found")
        return
    try:
        for rate in expired_rates:
            print(f"Sending notification for rate {rate.id}")
    except ServiceException as exc:
        raise self.retry(exc=exc) from exc
