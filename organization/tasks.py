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


from celery import shared_task
from django.core.management import call_command
from kombu.exceptions import OperationalError


@shared_task(bind=True)
def table_change_alerts(self) -> None:
    """
    A Celery task that listens for table change notifications from a PostgreSQL database and retries on errors.

    This task invokes the `psql_listener` management command using Django's `call_command` function, which sets up
    a PostgreSQL listener using the `psycopg2` library, and listens for notifications on the channels defined in
    the `TableChangeAlert` model. If an `OperationalError` is raised during the process, the task will retry the
    operation with exponential backoff.

    Args:
        self: A reference to the task instance.

    Returns:
        None.

    Raises:
        None.
    """
    try:
        call_command("psql_listener")
    except OperationalError as exc:
        raise self.retry(exc=exc) from exc