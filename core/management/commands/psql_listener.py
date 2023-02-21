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
