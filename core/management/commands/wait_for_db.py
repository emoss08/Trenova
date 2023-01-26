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

import time
from django.core.management.base import BaseCommand
from psycopg2 import OperationalError as Psycopg2OperationalError
from django.db.utils import OperationalError

class Command(BaseCommand):
    """
    Django command to pause execution until database is available.

    This command is used to check the availability of a database and waits for it to be ready. It uses the 'check' method
    provided by Django to check the availability of the 'default' database. If the database is not available, it waits
    for 'delay' seconds before retrying the connection. The delay time starts at 1 second, and is doubled in each
    iteration (up to a maximum of 60 seconds) so that the script doesn't retry the connection too frequently when the
    database is not available, which would put unnecessary load on the server. Once the database is available, the
    command prints a message 'Database available!' to the console.
    """
    def handle(self, *args: Any, **options: Any) -> None:
        """
        Handle the command.

        This method is called when the command is run. It writes the message 'Waiting for database...' to the console,
        and enters a loop to check the availability of the 'default' database using the 'check' method provided by Django.
        If the database is not available, it waits for 'delay' seconds before retrying the connection. Once the database
        becomes available, the method writes the message 'Database available!' to the console.

        Args:
            *args: Additional arguments passed to the command
            **options: Additional options passed to the command

        Returns:
            None
        """
        self.stdout.write('Waiting for database...')
        db_up = False
        delay = 1
        while not db_up:
            try:
                self.check(databases=['default'])
                db_up = True
            except (Psycopg2OperationalError, OperationalError):
                self.stdout.write(self.style.WARNING(f'Database unavailable, waiting {delay} second(s)...'))
                time.sleep(delay)
                delay = min(60, delay*2)

        self.stdout.write(self.style.SUCCESS('Database available!'))