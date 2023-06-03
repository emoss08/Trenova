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

from __future__ import annotations

import json
import os
import signal
from pathlib import Path
from typing import Any
import types
from confluent_kafka import Consumer, KafkaException, Message
from django.core.mail import send_mail
from django.db.models import QuerySet
from environ import environ
from organization import models, selectors

# Load environment variables
env = environ.Env()
ENV_DIR = Path(__file__).parent.parent
environ.Env.read_env(os.path.join(ENV_DIR, ".env"))


class KafkaListener:
    """Handles listening to a Kafka server for table change alerts.

    This class connects to a Kafka server and listens for messages in defined topics.
    These topics represent table changes in the database. The class also processes
    these messages and performs appropriate actions, like sending emails.

    Attributes:
        KAFKA_HOST (str): The hostname of the Kafka server to connect to.
        KAFKA_PORT (int): The port number of the Kafka server to connect to.
        KAFKA_GROUP_ID (str): The identifier for the Kafka consumer group.
        ALERT_UPDATE_TOPIC (str): The name of the Kafka topic where alert updates are published.
        POLL_TIMEOUT (float): The maximum amount of time in seconds the consumer will block waiting for message records to be available.
        NO_ALERTS_MSG (str): A string to display when there are no active table change alerts.
        interrupted (bool): A boolean to flag if the listener has been interrupted and should stop.
    """

    KAFKA_HOST = env("KAFKA_HOST")
    KAFKA_PORT = env("KAFKA_PORT")
    KAFKA_GROUP_ID = env("KAFKA_GROUP_ID")
    ALERT_UPDATE_TOPIC = "localhost.public.table_change_alert"
    POLL_TIMEOUT = 1.0
    NO_ALERTS_MSG = "No active table change alerts."
    interrupted = False

    # TODO(Wolfred): Replace all prints with SSE or websockets. Still haven't decided.

    @classmethod
    def signal_handler(cls, signal: int, frame: types.FrameType | None) -> None:
        """Handles a signal interruption.

        This method changes the 'interrupted' class variable to True if a signal interruption
        is received. This helps the listen method to know when to stop listening.

        Args:
            signal (int): The identifier of the received signal.
            frame (FrameType | None): The current stack frame.

        Returns:
            None: This method does not return anything.
        """
        print("Signal received, shutting down...")
        cls.interrupted = True

    @classmethod
    def connect(cls) -> tuple[Consumer, Consumer]:
        """Establishes connection with the Kafka server.

        This method initializes two Kafka Consumer instances using a common configuration.

        Returns:
            tuple[Consumer, Consumer]: A tuple containing two Consumer instances, one for data
            and one for alert updates.
        """
        config = {
            "bootstrap.servers": env("KAFKA_BOOTSTRAP_SERVERS"),
            "group.id": env("KAFKA_GROUP_ID"),
            "auto.offset.reset": "latest",
        }

        return Consumer(config), Consumer(config)

    @staticmethod
    def get_topic_list() -> QuerySet | list:
        """Retrieves the list of active Kafka table change alerts.

        This method queries the database for all active table change alerts and returns
        them as a QuerySet. If there are no active alerts, it returns an empty list.

        Returns:
            QuerySet[TableChangeAlert] | list: A queryset or list of TableChangeAlert instances with active alerts.
        """
        return selectors.get_active_kafka_table_change_alerts() or []

    @classmethod
    def get_message(cls, *, consumer: Consumer, timeout: float) -> Message:
        """Fetches a message from the Kafka consumer.

        This method polls the given Kafka consumer for a message, waiting for the specified timeout.

        Args:
            consumer (Consumer): The Kafka consumer instance from which to fetch the message.
            timeout (float): The maximum time to wait for a message.

        Returns:
            Message: The Kafka message instance if available, else None.
        """
        message = consumer.poll(timeout)
        if message is None:
            return None
        elif message.error():
            print(f"Consumer error: {message.error()}")
            return None
        return message

    @classmethod
    def update_subscriptions(
        cls,
        *,
        data_consumer: Consumer,
        table_changes: QuerySet | list,
    ) -> None:
        """Updates the topic subscription list of the data_consumer.

        This method compares the current list of table changes with a newly retrieved list.
        It then unsubscribes and subscribes the data_consumer to the new list of topics,
        if there are any changes.

        Args:
            data_consumer (Consumer): The Kafka consumer instance that needs its topic subscriptions updated.
            table_changes (QuerySet[TableChangeAlert] | list): The current list or queryset of TableChangeAlert instances.

        Returns:
            None: This method does not return anything.
        """
        old_table_changes = {
            table_change.get_topic_display() for table_change in table_changes
        }
        table_changes = cls.get_topic_list()
        new_table_changes = {
            table_change.get_topic_display() for table_change in table_changes
        }
        if added_alerts := new_table_changes.difference(old_table_changes):
            print(
                f"New alerts added: {', '.join(added_alerts)}",
            )
        data_consumer.unsubscribe()
        if table_changes:
            data_consumer.subscribe(
                [table_change.get_topic_display() for table_change in table_changes]
            )
            print(
                f"Subscribed to topics: {', '.join([table_change.get_topic_display() for table_change in table_changes])}"
            )

    @staticmethod
    def parse_message(*, message: Message) -> dict[str, Any]:
        """Parses a Kafka message.

        This method extracts the value from the Kafka message, decodes it from bytes to string,
        and converts it from JSON format to a Python dictionary.

        Args:
            message (Message): The Kafka message instance to parse.

        Returns:
            dict: The payload of the Kafka message as a dictionary.
        """
        message_value = message.value().decode("utf-8")
        data = json.loads(message_value)
        return data.get("payload", {})

    @staticmethod
    def format_message(*, field_value_dict: dict) -> str:
        """Formats a dictionary into a human-readable string message.

        This method takes a dictionary of field and value pairs and converts it into a string.
        Each key-value pair in the dictionary becomes a line in the string in the format "Field: <field>, Value: <value>".

        Args:
            field_value_dict (dict): A dictionary containing field-value pairs.

        Returns:
            str: A string representation of the field-value pairs in the dictionary.
        """
        return "\n".join(
            f"Field: {field}, Value: {value}"
            for field, value in field_value_dict.items()
        )

    @classmethod
    def process_message(
        cls, *, data_message: Message, associated_table_change: models.TableChangeAlert
    ) -> None:
        """Processes a Kafka message.

        This method takes a Kafka message and an associated TableChangeAlert model instance.
        It parses the message, checks if the type of operation in the message matches one
        in the TableChangeAlert instance, and sends an email alert if it does.

        Args:
            data_message (Message): The Kafka message instance to be processed.
            associated_table_change (TableChangeAlert): The TableChangeAlert instance associated with the topic of the message.

        Returns:
            None: This method does not return anything.
        """
        if not data_message.value():
            return

        data = cls.parse_message(message=data_message)

        op_type: str | None = data.get("op")

        op_type_mapping = {
            "c": models.TableChangeAlert.DatabaseActionChoices.INSERT,
            "u": models.TableChangeAlert.DatabaseActionChoices.UPDATE,
        }
        if not op_type:
            return
        translated_op_type = op_type_mapping.get(op_type)

        # If op_type is None or not in database_action, return immediately
        if (
            not translated_op_type
            or translated_op_type not in associated_table_change.database_action
        ):
            return

        field_value_dict = data.get("after") or {}

        recipient_list = (
            associated_table_change.email_recipients.split(",")
            if associated_table_change.email_recipients
            else []
        )
        subject = (
            associated_table_change.custom_subject
            or f"Table Change Alert: {data_message.topic()}"
        )

        print(f"Sending email to {recipient_list} with subject {subject}")
        send_mail(
            subject=subject,
            message=KafkaListener.format_message(field_value_dict=field_value_dict),
            from_email="table_change@monta.io",
            recipient_list=recipient_list,
        )

    @classmethod
    def listen(cls) -> None:
        """Starts the KafkaListener to listen to the Kafka server.

        This method sets up signal handlers, connects to the Kafka server, retrieves
        the list of active table change alerts, and subscribes the consumers to their
        respective topics. It then enters a loop where it listens for messages from
        both consumers, updates subscriptions when an alert update is received, and
        processes messages from the data consumer.

        Returns:
            None: This method does not return anything.
        """
        signal.signal(signal.SIGINT, cls.signal_handler)
        signal.signal(signal.SIGTERM, cls.signal_handler)
        data_consumer, alert_update_consumer = cls.connect()

        table_changes = cls.get_topic_list()
        if not table_changes:
            print(cls.NO_ALERTS_MSG)
            return

        alert_update_consumer.subscribe([cls.ALERT_UPDATE_TOPIC])
        print(f"Subscribed to alert update topic: {cls.ALERT_UPDATE_TOPIC}")
        data_consumer.subscribe(
            [table_change.get_topic_display() for table_change in table_changes]
        )
        print(
            f"Subscribed to topics: {[table_change.get_topic_display() for table_change in table_changes]}"
        )

        try:
            while True:
                if cls.interrupted:
                    print("Interrupt received, closing consumers...")
                    break

                alert_message = cls.get_message(
                    consumer=alert_update_consumer, timeout=cls.POLL_TIMEOUT
                )
                if alert_message is not None:
                    print(f"Received alert update: {alert_message.value()}")
                    cls.update_subscriptions(
                        data_consumer=data_consumer, table_changes=table_changes
                    )

                data_message = cls.get_message(
                    consumer=data_consumer, timeout=cls.POLL_TIMEOUT
                )
                if (
                    data_message is not None
                    and not data_message.error()
                    and data_message.value() is not None
                ):
                    print(
                        f"Received data: {data_message.value().decode('utf-8')} from topic: {data_message.topic()}"
                    )

                    # Getting TableChangeAlert object associated with the topic
                    associated_table_change = next(
                        (
                            table_change
                            for table_change in table_changes
                            if table_change.get_topic_display() == data_message.topic()
                        ),
                        None,
                    )
                    if associated_table_change and data_message:
                        print("Table Change Alert found.", associated_table_change.name)
                        cls.process_message(
                            data_message=data_message,
                            associated_table_change=associated_table_change,
                        )
        except KafkaException as e:
            print(f"Unexpected error: {e}")
        finally:
            data_consumer.close()
            alert_update_consumer.close()
