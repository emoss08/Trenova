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
import datetime

from celery import shared_task
from django.core.management import call_command
from django.utils import timezone


def get_cutoff_date() -> datetime.datetime:
    """Get the cutoff date for deleting audit log records.

    Returns:
    str: The cutoff date for deleting audit log records.
    """

    return timezone.now() - timezone.timedelta(days=30)


@shared_task
def delete_audit_log_records() -> str:
    """Delete audit log records older than 30 days.

    This task uses the Django management command `auditlogflush` to delete
    audit log records older than 30 days. The cutoff date is calculated by
    subtracting 30 days from the current date, and the `strftime` method is used
    to format the date in a usable format for the command.

    Returns:
    str: The message "Audit log records deleted." upon successful completion of the task.
    """

    cutoff_date = get_cutoff_date()
    formatted_date = cutoff_date.strftime("%Y-%m-%d")

    call_command("auditlogflush", "-b", formatted_date, "-y")

    return f"Successfully deleted audit log records. older than {formatted_date}."
