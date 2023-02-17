"""
COPYRIGHT 2023 MONTA

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

from __future__ import absolute_import

import datetime

from celery import shared_task
from django.core.management import call_command
from django.utils import timezone
from kombu.exceptions import OperationalError


def get_cutoff_date() -> datetime.datetime:
    """Get the cutoff date for deleting audit log records.

    Returns:
    str: The cutoff date for deleting audit log records.
    """

    return timezone.now() - timezone.timedelta(days=30)


@shared_task(bind=True)
def delete_audit_log_records(self) -> str:
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

    cutoff_date: datetime.datetime = get_cutoff_date()
    formatted_date: str = cutoff_date.strftime("%Y-%m-%d")

    try:
        call_command("auditlogflush", "-b", formatted_date, "-y")
    except OperationalError as exc:
        raise self.retry(exc=exc) from exc

    return f"Successfully deleted audit log records. older than {formatted_date}."
