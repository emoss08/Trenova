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

import concurrent
import json
import logging
import os
import signal
import time
import types
from concurrent.futures import ThreadPoolExecutor, wait
from pathlib import Path
from typing import Any

from confluent_kafka import Consumer, KafkaError, KafkaException, Message
from django.core.mail import send_mail
from django.db.models import QuerySet
from environ import environ

from organization import models, selectors

# Load environment variables
env = environ.Env()
ENV_DIR = Path(__file__).parent.parent
environ.Env.read_env(os.path.join(ENV_DIR, ".env"))

# Logging Configuration
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s - %(name)s - %(levelname)s - %(message)s",
    handlers=[logging.FileHandler("kafka_listener.log"), logging.StreamHandler()],
)


class KafkaListener:
    """`KafkaListener` is a class that provides functionality to listen to specific Kafka topics
    and process their messages accordingly. It is primarily responsible for handling table change alerts.

    Class Attributes:
        KAFKA_HOST (str): The hostname or IP address of the Kafka service to connect to.
        KAFKA_PORT (str): The port number on which the Kafka service is running.
        KAFKA_GROUP_ID (str): The unique group identifier for this Kafka consumer instance.
        ALERT_UPDATE_TOPIC (str): The Kafka topic that this listener instance subscribes to for alert updates.
        POLL_TIMEOUT (float): The time in seconds that the listener should wait for a response from the Kafka service.
        NO_ALERTS_MSG (str): The default message to display when there are no active alerts.
        running (bool): A flag that indicates whether the listener instance is currently running.
        MAX_CONCURRENT_JOBS (int): The maximum number of jobs that this listener instance can process concurrently.
    """

    KAFKA_HOST = env("KAFKA_HOST")
    KAFKA_PORT = env("KAFKA_PORT")
    KAFKA_GROUP_ID = env("KAFKA_GROUP_ID")
    ALERT_UPDATE_TOPIC = "localhost.public.table_change_alert"
    POLL_TIMEOUT = 1.0
    NO_ALERTS_MSG = "No active table change alerts."
    running = True
    MAX_CONCURRENT_JOBS = 100
    THREAD_POOL_SIZE = env("THREAD_POOL_SIZE", default=10)

    # TODO(Wolfred): Replace all prints with SSE or websockets. Still haven't decided.

    def __repr__(self) -> str:
        """A built-in function that provides a string representation of the KafkaListener instance.

        Returns:
            str: The string representation includes the host, port, and group id of the Kafka consumer.
        """

        return f"KafkaListener({self.KAFKA_HOST}, {self.KAFKA_PORT}, {self.KAFKA_GROUP_ID})"

    def __init__(self, thread_pool_size=10) -> None:
        self.thread_pool_size = thread_pool_size

    @classmethod
    def _signal_handler(cls, _signal: int, frame: types.FrameType | None) -> None:
        """A signal handler method that sets the `running` attribute to False on receiving a termination signal,
        thereby controlling the runtime of the listener.

        Args:
            _signal (int): The identification number of the received signal.
            frame (types.FrameType | None): The current stack frame (relevant for traceback information).

        Returns:
            None: This function does not return anything.
        """

        logging.info("Received termination signal. Stopping listener...")
        cls.running = False

    @classmethod
    def _connect(cls) -> tuple[Consumer, Consumer] | None:
        """A private method to establish a connection to the Kafka service.
        In case of connection failure, it retries until the listener stops running or a connection is established.

        Returns:
            tuple[Consumer, Consumer] | None: Returns a tuple of Consumer instances if connection is successful, else None.
        """

        config = {
            "bootstrap.servers": env("KAFKA_BOOTSTRAP_SERVERS"),
            "group.id": env("KAFKA_GROUP_ID"),
            "auto.offset.reset": "latest",
            "enable.auto.commit": "False",
            "fetch.min.bytes": 50000,
            "auto.commit.interval.ms": 5000,
        }

        while cls.running:
            try:
                consumer = Consumer(config)
                consumer.list_topics(timeout=10)
                return consumer, Consumer(config)
            except KafkaError as e:
                if e.args[0].code() != KafkaError._ALL_BROKERS_DOWN:
                    logging.error(f"KafkaError: {e}")
                    raise e
                logging.info("All brokers are down. Retrying connection...")
                time.sleep(5)
        return None

    @staticmethod
    def _get_topic_list() -> QuerySet | list:
        """Fetches the list of currently active Kafka table change alerts from the database.

        Returns:
            QuerySet | list: Returns a QuerySet or list of active Kafka table change alerts.
        """

        return selectors.get_active_kafka_table_change_alerts() or []

    @classmethod
    def _get_messages(
        cls, *, consumer: Consumer, timeout: float, max_messages: int = 100
    ) -> list[Message]:
        """Consumes a batch of messages from the Kafka topic within the specified timeout.
        It filters out messages that are None or contain errors.

        Args:
            consumer (Consumer): Kafka Consumer instance to consume messages.
            timeout (float): Maximum time, in seconds, to block waiting for a message.
            max_messages (int, optional): Maximum number of messages to return. Defaults to 100.

        Returns:
            list[Message]: List of valid Kafka Message instances.
        """

        messages = consumer.consume(max_messages, timeout)
        valid_messages = []
        for message in messages:
            if message is None:
                continue
            elif message.error():
                logging.error(f"Consumer error: {message.error()}")
                continue
            valid_messages.append(message)
        return valid_messages

    @staticmethod
    def _parse_message(*, message: Message) -> dict[str, Any] | None:
        """Parses the JSON payload of a Kafka message. In case the message value can't be decoded as JSON,
        it logs an error message and returns None.

        Args:
            message (Message): Kafka Message instance to parse.

        Returns:
            dict[str, Any] | None: The parsed JSON data as dictionary if the message is valid, else None.
        """

        message_value = message.value().decode("utf-8")
        try:
            data = json.loads(message_value)
        except json.JSONDecodeError:
            logging.error("Error decoding message value as JSON.")
            return None
        return data.get("payload", {})

    @classmethod
    def _get_message(cls, *, consumer: Consumer, timeout: float) -> Message:
        """Fetches a single message from the Kafka topic within the specified timeout.

        Args:
            consumer (Consumer): Kafka Consumer instance to consume the message.
            timeout (float): Maximum time, in seconds, to block waiting for a message.

        Returns:
            Message: A Kafka Message instance.
        """

        message = consumer.poll(timeout)
        if message is None:
            return None
        elif message.error():
            logging.error(f"Consumer error: {message.error()}")
            return None
        return message

    @classmethod
    def _update_subscriptions(
        cls,
        *,
        data_consumer: Consumer,
        table_changes: QuerySet | list,
    ) -> None:
        """Updates the subscription list of the Kafka Consumer to include the topics specified in the table_changes.
        It unsubscribes from any topics not present in the table_changes.

        Args:
            data_consumer (Consumer): Kafka Consumer instance.
            table_changes (QuerySet | list): A QuerySet or list containing the updated list of Kafka topics.

        Returns:
            None: This function does not return anything.
        """

        old_table_changes = {
            table_change.get_topic_display() for table_change in table_changes
        }
        table_changes = cls._get_topic_list()
        new_table_changes = {
            table_change.get_topic_display() for table_change in table_changes
        }
        if added_alerts := new_table_changes.difference(old_table_changes):
            logging.info(f"New alerts added: {', '.join(added_alerts)}")
        data_consumer.unsubscribe()
        if table_changes:
            data_consumer.subscribe(
                [table_change.get_topic_display() for table_change in table_changes]
            )
            logging.info(
                f"Subscribed to topics: {', '.join([table_change.get_topic_display() for table_change in table_changes])}"
            )

    @staticmethod
    def _format_message(*, field_value_dict: dict) -> str:
        """Formats the Kafka message fields and their corresponding values into a human-readable string.

        Args:
            field_value_dict (dict): Dictionary where keys are field names and values are corresponding field values.

        Returns:
            Text: String representation of each field and its corresponding value, each on a new line.
        """

        return "\n".join(
            f"Field: {field}, Value: {value}"
            for field, value in field_value_dict.items()
        )

    @classmethod
    def _process_message(
        cls, *, data_message: Message, associated_table_change: models.TableChangeAlert
    ) -> None:
        """Processes an individual message from a Kafka topic. If the operation type matches the alert criteria,
        it sends an email to the designated recipients.

        Args:
            data_message (Message): Kafka Message instance to process.
            associated_table_change (models.TableChangeAlert): The table change alert associated with the topic of the message.
        """

        if not data_message.value():
            return

        data = cls._parse_message(message=data_message)

        if data is None:  # Added to handle cases where message is not valid JSON.
            return

        op_type = data.get("op")

        op_type_mapping = {
            "c": models.TableChangeAlert.DatabaseActionChoices.INSERT,
            "u": models.TableChangeAlert.DatabaseActionChoices.UPDATE,
        }
        if not op_type:
            return
        translated_op_type = op_type_mapping.get(op_type)

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
        logging.info(
            f"Sending email to {recipient_list} with subject {subject} for message {data_message}"
        )
        send_mail(
            subject=subject,
            message=cls._format_message(field_value_dict=field_value_dict),
            from_email="table_change@monta.io",
            recipient_list=recipient_list,
        )

    @classmethod
    def listen(cls) -> None:
        """Initiates the Kafka listener. It establishes a connection to the Kafka service, subscribes to the necessary topics,
        and begins processing messages. This method runs indefinitely until the listener receives a termination signal.
        It also handles exceptions due to lost connection to the Kafka service by attempting to reconnect.

        Returns:
            None: This function does not return anything.
        """

        signal.signal(signal.SIGINT, cls._signal_handler)
        signal.signal(signal.SIGTERM, cls._signal_handler)
        consumers = cls._connect()

        if consumers is None:
            logging.error("Failed to connect, exiting...")
            return

        data_consumer, alert_update_consumer = consumers

        table_changes = cls._get_topic_list()
        if not table_changes:
            logging.info(cls.NO_ALERTS_MSG)
            return

        alert_update_consumer.subscribe([cls.ALERT_UPDATE_TOPIC])
        logging.info(f"Subscribed to alert update topic: {cls.ALERT_UPDATE_TOPIC}")
        data_consumer.subscribe(
            [table_change.get_topic_display() for table_change in table_changes]
        )
        logging.info(
            f"Subscribed to topics: {[table_change.get_topic_display() for table_change in table_changes]}"
        )

        futures = set()
        with ThreadPoolExecutor(max_workers=cls.THREAD_POOL_SIZE) as executor:
            try:
                while cls.running:
                    # Backpressure mechanism. If there are too many ongoing tasks, stop pulling in new messages.
                    if len(futures) < cls.MAX_CONCURRENT_JOBS:
                        alert_message = cls._get_message(
                            consumer=alert_update_consumer, timeout=cls.POLL_TIMEOUT
                        )

                        if alert_message is not None:
                            logging.info(
                                f"Received alert update: {alert_message.value()}"
                            )
                            cls._update_subscriptions(
                                data_consumer=data_consumer, table_changes=table_changes
                            )

                            table_changes = cls._get_topic_list()

                        data_messages = cls._get_messages(
                            consumer=data_consumer, timeout=cls.POLL_TIMEOUT
                        )

                        for data_message in data_messages:
                            if (
                                data_message is not None
                                and not data_message.error()
                                and data_message.value() is not None
                            ):
                                logging.info(
                                    f"Received data: {data_message.value().decode('utf-8')} from topic: {data_message.topic()}"
                                )

                                associated_table_change = next(
                                    (
                                        table_change
                                        for table_change in table_changes
                                        if table_change.get_topic_display()
                                        == data_message.topic()
                                    ),
                                    None,
                                )

                                if associated_table_change and data_message:
                                    logging.info(
                                        f"Table Change Alert found. {associated_table_change.name}"
                                    )
                                    future = executor.submit(
                                        cls._process_message,
                                        data_message=data_message,
                                        associated_table_change=associated_table_change,
                                    )
                                    futures.add(future)

                    # Wait for at least one of the futures to complete if there's too many of them.
                    if len(futures) >= cls.MAX_CONCURRENT_JOBS:
                        done, futures = wait(
                            futures, return_when=concurrent.futures.FIRST_COMPLETED
                        )

                    # Always try to get completed futures and remove them from the set of futures.
                    done, futures = concurrent.futures.wait(
                        futures, return_when=concurrent.futures.FIRST_COMPLETED
                    )
                    for future in done:
                        try:
                            future.result()
                        except Exception as e:
                            logging.info(
                                f"Error processing message: {e}", exc_info=True
                            )

                    data_consumer.commit(asynchronous=True)
                    futures = {f for f in futures if not f.done()}

            except KafkaException as e:
                if e.args[0].code() != KafkaError._ALL_BROKERS_DOWN:
                    raise e
                logging.error(
                    "All brokers are down. Attempting to reconnect...", exc_info=True
                )
                data_consumer, alert_update_consumer = cls._connect()

            finally:
                alert_update_consumer.close()
                data_consumer.close()
                logging.info("Consumers closed.")
