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

from backend.celery import app
from core.exceptions import CommandCallException
from django.core.management import call_command

if TYPE_CHECKING:
    from celery.app.task import Task


@app.task(name="table_change_alerts", bind=True, max_retries=3, default_retry_delay=60)
def table_change_alerts(self: "Task") -> None:
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
    except CommandCallException as exc:
        raise self.retry(exc=exc) from exc
