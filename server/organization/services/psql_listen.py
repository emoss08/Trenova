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
import selectors
import sys

import environ
from django.db.models import QuerySet
from psycopg import Cursor, OperationalError, connect, sql

from organization import models
from organization.selectors import get_active_psql_table_change_alerts

# Configure logger for 'psql_listener'
logger = logging.Logger("psql_listener")
console_handler = logging.StreamHandler(sys.stdout)
console_handler.setLevel(logging.INFO)

formatter = logging.Formatter("%(asctime)s - %(name)s - %(levelname)s - %(message)s")
console_handler.setFormatter(formatter)

logger.addHandler(console_handler)

env = environ.Env()


class PSQLListener:
    def __init__(self):
        self.table_change_alert_channel = "table_change_alert_updated"
        self.conn = None

    def ensure_trigger_exists(self) -> None:
        try:
            with self.conn.cursor() as cur:
                self._create_trigger_function_if_not_exists(cur=cur)
                self._create_trigger_if_not_exists(cur=cur)
        except Exception as e:
            logger.error(f"Error ensuring trigger exists: {e}")
            raise

    @staticmethod
    def _create_trigger_function_if_not_exists(*, cur: Cursor) -> None:
        trigger_function_name = "notify_table_change"
        cur.execute(
            "SELECT 1 FROM pg_proc WHERE proname = %s;", (trigger_function_name,)
        )
        if not cur.fetchone():
            logger.info(f"Function {trigger_function_name} does not exist. Creating...")
            cur.execute(
                """
                CREATE OR REPLACE FUNCTION notify_table_change() RETURNS TRIGGER AS $$
                BEGIN
                    PERFORM pg_notify('table_change_alert_updated', TG_TABLE_NAME || ' ' || TG_OP);
                    RETURN NEW;
                END;
                $$ LANGUAGE plpgsql;
            """
            )
            logger.info(f"Function {trigger_function_name} created.")

    @staticmethod
    def _create_trigger_if_not_exists(*, cur: Cursor) -> None:
        trigger_name = "table_change_alert_trigger"
        cur.execute(
            sql.SQL("SELECT 1 FROM pg_trigger WHERE tgname = %s;"), [trigger_name]
        )
        if not cur.fetchone():
            logger.info(f"Trigger {trigger_name} does not exist. Creating...")
            table_name = "public.table_change_alert"
            trigger_function_name = "notify_table_change"
            cur.execute(
                sql.SQL(
                    """
                    CREATE TRIGGER {trigger_name}
                    AFTER INSERT OR UPDATE OR DELETE ON {table}
                    FOR EACH ROW EXECUTE PROCEDURE {function_name}();
                """
                ).format(
                    trigger_name=sql.Identifier(trigger_name),
                    table=sql.Identifier(table_name),
                    function_name=sql.Identifier(trigger_function_name),
                )
            )
            logger.info(f"Trigger {trigger_name} created.")

    def connect(self) -> None:
        self.conn = connect(
            dbname=env("DB_NAME"),
            user=env("DB_USER"),
            password=env("DB_PASSWORD"),
            host=env("DB_HOST"),
            port=env("DB_PORT"),
            autocommit=True,
        )

    def listen(self) -> None:
        self.connect()
        try:
            self.ensure_trigger_exists()
            self._start_listening()
        except Exception as e:
            logger.error(f"Error during listening: {e}")
        finally:
            if self.conn:
                self.conn.close()

    def _start_listening(self) -> None:
        table_changes = get_active_psql_table_change_alerts()

        if not table_changes:
            logger.warning("No active table change alerts.")
            return

        with self.conn.cursor() as cur:
            self._listen_to_channels(cur, table_changes)
            self._poll_notifications()

    def _listen_to_channels(
        self, cur: Cursor, table_changes: QuerySet[models.TableChangeAlert]
    ) -> None:
        for change in table_changes:
            if change.listener_name:
                self._execute_listen(cur=cur, channel_name=change.listener_name)
            else:
                logger.warning(f"Listener name is empty for change: {change}")

        self._execute_listen(cur=cur, channel_name=self.table_change_alert_channel)

    @staticmethod
    def _execute_listen(*, cur: Cursor, channel_name: str) -> None:
        listen_query = sql.SQL("LISTEN {}").format(sql.Identifier(channel_name))
        cur.execute(listen_query)
        logger.info(f"Listening to channel: {channel_name}")

    def _poll_notifications(self) -> None:
        sel = selectors.DefaultSelector()
        sel.register(self.conn, selectors.EVENT_READ)

        while True:
            if not sel.select(timeout=60.0):
                # No FD activity detected in one minute
                continue

            # Activity detected. Is the connection still ok?
            try:
                self.conn.execute("SELECT 1")
                self._handle_notifications()
            except OperationalError:
                # You were disconnected: handle this case, such as re-establishing connection or exiting
                logger.error("Lost connection to the database!")
                sys.exit(1)

    def _handle_notifications(self) -> None:
        # Iterate over the generator of notifications
        for notify in self.conn.notifies():
            logger.info(f"Got NOTIFY: {notify.pid}, {notify.channel}, {notify.payload}")
            if notify.channel == self.table_change_alert_channel:
                self._restart_listening(self.conn.cursor())

    def _restart_listening(self, cur: Cursor) -> None:
        cur.execute(sql.SQL("UNLISTEN *;"))
        logger.info("Restarting listener due to new TableChangeAlert...")
        table_changes = get_active_psql_table_change_alerts()
        self._listen_to_channels(cur, table_changes)
