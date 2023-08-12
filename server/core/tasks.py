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

import datetime
from typing import TYPE_CHECKING

from backend.celery import app
from celery import shared_task
from core.exceptions import CommandCallException
from django.core.management import call_command
from django.utils import timezone

if TYPE_CHECKING:
    from celery.app.task import Task


@app.task(
    name="delete_audit_log_records", bind=True, max_retries=3, default_retry_delay=60
)
def delete_audit_log_records(self: "Task") -> str:
    """Delete audit log records older than 30 days.

    This task uses the Django management command `auditlogflush` to delete
    audit log records older than 30 days. The cutoff date is calculated by
    subtracting 30 days from the current date, and the `strftime` method is used
    to format the date in a usable format for the command.

    Args:
        self (celery.app.task.Task): The task object

    Returns:
        str: The message "Audit log records deleted." upon successful completion of the task.
    """

    cutoff_date = timezone.now() - datetime.timedelta(days=30)
    formatted_date = cutoff_date.strftime("%Y-%m-%d")

    try:
        call_command("auditlogflush", "-b", formatted_date, "-y")
    except CommandCallException as exc:
        raise self.retry(exc=exc) from exc

    return f"Successfully deleted audit log records. older than {formatted_date}."


@shared_task(ignore_result=False)
def add(x: int, y: int) -> int:
    return x + y
