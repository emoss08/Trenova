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

from django.core.management.base import BaseCommand
from organization.services.psql_listen import PSQLListener


class Command(BaseCommand):
    """
    A Django management command that listens for PostgreSQL notifications and logs them to the console.

    This command sets up a PostgreSQL listener using the `psycopg2` library, and listens for notifications on
    the channels defined in the `TableChangeAlert` model. When a notification is received, it is logged to the
    console using Django's built-in logging system.

    Example usage:
        python manage.py psql_listener

    Returns:
        None.
    """

    help = "Listens for PostgreSQL notifications and logs them to the console."

    def handle(self, *args: Any, **options: Any) -> None:
        """
        Runs the main body of the command.

        This method invokes the `psql_listener` function, which sets up a PostgreSQL listener using the
        `psycopg2` library, and listens for notifications on the channels defined in the `TableChangeAlert`
        model. When a notification is received, it is logged to the console using Django's built-in logging system.

        Args:
            args: A list of positional arguments.
            options: A dictionary of keyword arguments.

        Returns:
            None.

        Raises:
            None.
        """

        self.stdout.write(self.style.SUCCESS("Starting PostgreSQL listener..."))

        listener = PSQLListener()
        listener.listen()
