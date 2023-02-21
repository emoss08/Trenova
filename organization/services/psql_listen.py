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

import os
from pathlib import Path

import psycopg2
from environ import environ

from organization.selectors import get_active_table_alerts

env = environ.Env()
ENV_DIR = Path(__file__).parent.parent.parent
environ.Env.read_env(os.path.join(ENV_DIR, ".env"))


class PSQLListener:
    """
    Listens for table change notifications from a PostgreSQL database and prints them to the console.

    This class provides a `listen()` method that retrieves a list of active table change alerts from the database
    using the `get_active_table_alerts()` function. It then sets up a PostgreSQL connection using the `connect()`
    method, and registers a PostgreSQL listener for each alert. When a notification is received on a channel, the
    `listen()` method prints the notification details to the console.

    The `connect()` method is a class method that creates a connection to the PostgreSQL database using the
    `psycopg2.connect()` function, and sets the `autocommit` attribute to `True`.

    Example usage:
    >>>    listener = PSQLListener()
    >>>    listener.listen()
    """

    @classmethod
    def connect(cls):
        """
        Creates a connection to a PostgreSQL database.

        Returns:
            psycopg2.connection: A connection to the PostgreSQL database.

        Raises:
            None.
        """
        conn = psycopg2.connect(
            host="localhost",
            database=env("DB_NAME"),
            user=env("DB_USER"),
            password=env("DB_PASSWORD"),
            port=5432,
        )
        conn.autocommit = True
        return conn

    @classmethod
    def listen(cls) -> None:
        """
        Listens for table change notifications and prints them to the console.

        This method retrieves a list of active table change alerts from the database using the
        `get_active_table_alerts()` function. It then sets up a PostgreSQL connection using the `connect()`
        method, and registers a PostgreSQL listener for each alert. When a notification is received on a channel,
        the method prints the notification details to the console.

        Returns:
            None.

        Raises:
            None.
        """
        conn = cls.connect()
        table_changes = get_active_table_alerts()

        if not table_changes:
            print("No active table change alerts.")
            conn.close()
            return

        with conn.cursor() as cur:
            for change in table_changes:
                cur.execute(f"LISTEN {change.listener_name};")

            while True:
                conn.poll()
                while conn.notifies:
                    notify = conn.notifies.pop(0)
                    print(
                        f"Got NOTIFY: {notify.pid}, {notify.channel}, {notify.payload}"
                    )