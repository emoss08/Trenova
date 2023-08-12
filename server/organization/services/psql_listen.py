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

import logging
import os
import select
from pathlib import Path

import psycopg2
from environ import environ
from organization.selectors import get_active_table_alerts

# Logging Configuration
logger = logging.getLogger(__name__)

# Environment Configuration
env = environ.Env()
ENV_DIR = Path(__file__).parent.parent.parent
environ.Env.read_env(os.path.join(ENV_DIR, ".env"))


class PSQLListener:
    """A class representing a PostgreSQL listener for table change alerts.

    This class provides methods to connect to a PostgreSQL database and
    listen to notifications on specific channels. It sets up listeners for
    table change alerts and handles notifications by printing them.

    Methods:
        connect() -> psycopg2.extensions.connection:
            Establishes a connection to a PostgreSQL database using psycopg2.
        listen() -> None:
            Sets up listeners for table change alerts and handles notifications.
    """

    @classmethod
    def ensure_trigger_exists(cls, conn: psycopg2.extensions.connection) -> None:
        """Ensures that a specific trigger exists on a given table.

        Args:
            conn(connection): A psycopg2.extensions.connection instance.

        Returns:
            None: This function does not return anything.
        """
        trigger_name = "table_change_alert_trigger"
        trigger_function_name = "notify_table_change"
        table_name = "public.table_change_alert"

        with conn.cursor() as cur:
            # Check if the function exists
            cur.execute(
                """
                SELECT 1
                FROM pg_proc
                WHERE proname = %s;
            """,
                (trigger_function_name,),
            )
            function_exists = cur.fetchone()

            # Create the function if it doesn't exist
            if not function_exists:
                logger.info(
                    f"Function {trigger_function_name} does not exist. Creating..."
                )
                cur.execute(
                    f"""
                    CREATE OR REPLACE FUNCTION {trigger_function_name}() RETURNS TRIGGER AS $$
                    BEGIN
                        PERFORM pg_notify('table_change_alert_updated', TG_TABLE_NAME || ' ' || TG_OP);
                        RETURN NEW;
                    END;
                    $$ LANGUAGE plpgsql;
                """
                )
                logger.info(f"Function {trigger_function_name} created.")

            # Check if the trigger exists
            cur.execute(
                """
                SELECT 1
                FROM pg_trigger
                WHERE tgname = %s;
            """,
                (trigger_name,),
            )
            trigger_exists = cur.fetchone()

            # Create the trigger if it doesn't exist
            if not trigger_exists:
                logger.info(f"Trigger {trigger_name} does not exist. Creating...")
                cur.execute(
                    f"""
                    CREATE TRIGGER {trigger_name}
                    AFTER INSERT OR UPDATE OR DELETE ON {table_name}
                    FOR EACH ROW EXECUTE PROCEDURE {trigger_function_name}();
                """
                )
                logger.info(f"Trigger {trigger_name} created.")

    @classmethod
    def connect(cls) -> psycopg2.extensions.connection:
        """Connect to a PostgreSQL database using psycopg2.

        This method reads database connection information from environment
        variables and returns a connection to the specified database.

        Returns:
            psycopg2.connection: A connection to the PostgreSQL database.
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
        """Set up listeners for table change alerts and handle notifications.

        This method connects to the database, sets up listeners for table
        change alerts, and handles notifications by printing them. If a
        notification is received on the table_change_alert_channel, it restarts
        the listeners for table_changes.

        Returns:
            None: This function does not return anything.
        """
        conn = cls.connect()
        cls.ensure_trigger_exists(conn)
        table_changes = get_active_table_alerts()
        table_change_alert_channel = "table_change_alert_updated"

        if not table_changes:
            logger.warning("No active table change alerts.")
            conn.close()
            return

        with conn.cursor() as cur:
            for change in table_changes:
                cur.execute("LISTEN %s;", (change.listener_name,))
                logger.info(f"Listening to channel: {change.listener_name}")

            cur.execute("LISTEN %s;", (table_change_alert_channel,))
            logger.info(f"Listening to channel: {table_change_alert_channel}")

            while True:
                rlist, _, _ = select.select([conn.fileno()], [], [], 5)
                if conn.fileno() in rlist:
                    conn.poll()
                    while conn.notifies:
                        notify = conn.notifies.pop(0)
                        logger.info(
                            f"Got NOTIFY: {notify.pid}, {notify.channel}, {notify.payload}"
                        )

                        if notify.channel == table_change_alert_channel:
                            cur.execute("UNLISTEN *;")
                            logger.info(
                                "Restarting listener due to new TableChangeAlert..."
                            )
                            for change in table_changes:
                                cur.execute("LISTEN %s;", (change.listener_name,))
                                logger.info(
                                    f"Listening to channel: {change.listener_name}"
                                )

                            cur.execute("LISTEN %s;", (table_change_alert_channel,))
                else:
                    logger.info("Timeout reached, no notifications received.")
