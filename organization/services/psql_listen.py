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
