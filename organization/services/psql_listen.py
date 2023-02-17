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

conn = psycopg2.connect(
    host=env("DB_HOST"),
    database=env("DB_NAME"),
    user=env("DB_USER"),
    password=env("DB_PASSWORD"),
    port=env("DB_PORT"),
)


def pgsql_listener() -> str | None:
    """
    Listens for table change notifications from the PostgreSQL database and prints them to the console.

    This function retrieves a list of active TableChangeAlert objects from the database, and sets up a
    PostgreSQL connection using the psycopg2 library. It then iterates over the list of alerts and
    registers a PostgreSQL listener for each one, using the alert's 'listener_name' attribute as the
    channel name. When a notification is received on a channel, the function prints the notification
    details to the console.

    Returns:
        str | None: A string indicating that no active table change alerts were found, or None if
        active alerts were found.

    Raises:
        None.

    Example usage:
        psql_listener()
    """
    table_changes = get_active_table_alerts()

    if not table_changes:
        return "No active table change alerts."

    with conn.cursor() as cur:
        for change in table_changes:
            cur.execute(f"LISTEN {change.listener_name};")
            conn.commit()

        while True:
            conn.poll()
            while conn.notifies:
                notify = conn.notifies.pop(0)
                print("Got NOTIFY:", notify.pid, notify.channel, notify.payload)
