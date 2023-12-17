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

import time
from typing import Any

from django.core.management.base import BaseCommand
from django.db.utils import OperationalError
from psycopg import OperationalError as PsycopgOperationalError


class Command(BaseCommand):
    """
    Django command to pause execution until database is available.

    This command is used to check the availability of a database and waits for it to be ready. It uses the
    'check' method provided by Django to check the availability of the 'default' database. If the database
    is not available, it waits for 'delay' seconds before retrying the connection. The delay time starts
    at 1 second, and is doubled in each iteration (up to a maximum of 60 seconds) so that the script
    doesn't retry the connection too frequently when the database is not available, which would put
    unnecessary load on the server. Once the database is available, the command prints a message
    'Database available!' to the console.
    """

    def handle(self, *args: Any, **options: Any) -> None:
        """
        Handle the command.

        This method is called when the command is run. It writes the message 'Waiting for database...' to the
        console, and enters a loop to check the availability of the 'default' database using the 'check'
        method provided by Django. If the database is not available, it waits for 'delay' seconds
        before retrying the connection. Once the database becomes available, the method writes the
        message 'Database available!' to the console.

        Args:
            *args: Additional arguments passed to the command
            **options: Additional options passed to the command

        Returns:
            None
        """
        self.stdout.write("Waiting for database...")
        db_up = False
        delay = 1
        while not db_up:
            try:
                self.check(databases=["default"])
                db_up = True
            except (PsycopgOperationalError, OperationalError):
                self.stdout.write(
                    self.style.WARNING(
                        f"Database unavailable, waiting {delay} second(s)..."
                    )
                )
                time.sleep(delay)
                delay = min(60, delay * 2)

        self.stdout.write(self.style.SUCCESS("Database available!"))
