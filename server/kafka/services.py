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
import concurrent.futures
import json
import logging
import signal
import time
import types
from typing import Any

from confluent_kafka import Consumer, KafkaError, KafkaException, Message
from django.conf import settings
from django.core.mail import send_mail
from django.db import connections
from django.db.models import QuerySet

from organization import models, selectors

# Logging Configuration
logger = logging.getLogger("kafka")

debug, error = logger.debug, logger.error


POLL_TIMEOUT = 1.0
NO_ALERTS_MSG = "No active table change alerts."


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

    running = True
    # TODO(Wolfred): Replace all prints with SSE or websockets. Still haven't decided.

    @classmethod
    def _close_old_connections(cls, **kwargs: Any) -> None:
        """A method that closes all old database connections.

        Args:
            **kwargs (Any): Any additional keyword arguments.

        Returns:
            None: This function does not return anything.
        """
        for conn in connections.all(initialized_only=True):
            conn.close_if_unusable_or_obsolete()

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

        debug("Received termination signal. Stopping listener...")
        cls.running = False

    @classmethod
    def _connect(cls) -> tuple[Consumer, Consumer] | None:
        """A private method to establish a connection to the Kafka service.
        In case of connection failure, it retries until the listener stops running or a connection is established.

        Returns:
            tuple[Consumer, Consumer] | None: Returns a tuple of Consumer instances if connection is successful, else None.
        """

        config = {
            "bootstrap.servers": settings.KAFKA_BOOTSTRAP_SERVERS,
            "group.id": settings.KAFKA_GROUP_ID,
            "auto.offset.reset": settings.KAFKA_AUTO_OFFSET_RESET,
            "enable.auto.commit": settings.KAFKA_AUTO_COMMIT,
            "fetch.min.bytes": settings.KAFKA_AUTO_COMMIT_INTERVAL_MS,
            "auto.commit.interval.ms": settings.KAFKA_AUTO_COMMIT_INTERVAL_MS,
        }

        while cls.running:
            try:
                consumer = Consumer(config)
                consumer.list_topics(timeout=10)
                return consumer, Consumer(config)
            except KafkaError as e:
                if e.args[0].code() != KafkaError._ALL_BROKERS_DOWN:
                    error(f"KafkaError: {e}")
                    raise e
                debug("All brokers are down. Retrying connection...")
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
                error(f"Consumer error: {message.error()}")
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
            error("Error decoding message value as JSON.")
            return None
        return data.get("payload", {})

    @classmethod
    def _get_message(cls, *, consumer: Consumer, timeout: float) -> Message | None:
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
            error(f"Consumer error: {message.error()}")
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
            debug(f"New alerts added: {', '.join(added_alerts)}")
        data_consumer.unsubscribe()
        if table_changes:
            data_consumer.subscribe(
                [table_change.get_topic_display() for table_change in table_changes]
            )
            debug(
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
        cls, data_message: Message, associated_table_change: models.TableChangeAlert
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
        debug(
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
        """Entry point for Kafka listener. It establishes a connection, subscribes to the necessary topics,
        and begins processing messages. This method runs indefinitely until the listener receives a termination signal.
        It also handles exceptions due to lost connection to the Kafka service by attempting to reconnect.

        Returns:
            None: This function does not return anything.
        """
        cls.register_signals()
        consumers = cls._connect()

        if consumers is None:
            error("Failed to connect, exiting...")
            return

        data_consumer, alert_update_consumer = consumers
        cls._subscribe_consumers_to_topics(
            data_consumer=data_consumer, alert_update_consumer=alert_update_consumer
        )

        cls._execute_tasks(
            data_consumer=data_consumer, alert_update_consumer=alert_update_consumer
        )

        alert_update_consumer.close()
        data_consumer.close()
        debug("Consumers closed.")

    @classmethod
    def register_signals(cls) -> None:
        """Register signals for termination. These signals are used to terminate the Kafka listener gracefully.

        Returns:
            None: This function does not return anything.
        """
        signal.signal(signal.SIGINT, cls._signal_handler)
        signal.signal(signal.SIGTERM, cls._signal_handler)

    @classmethod
    def _subscribe_consumers_to_topics(
        cls, *, data_consumer: Consumer, alert_update_consumer: Consumer
    ) -> None:
        """
        Subscribe the provided consumers to the necessary topics. The topics are fetched from the _get_topic_list method.

        Args:
            data_consumer (Consumer): Consumer for data messages.
            alert_update_consumer (Consumer): Consumer for alert update messages.

        Returns:
            None: This function does not return anything.
        """
        table_changes = cls._get_topic_list()
        if not table_changes:
            debug(NO_ALERTS_MSG)
            return

        alert_update_consumer.subscribe([settings.KAFKA_ALERT_UPDATE_TOPIC])
        debug(f"Subscribed to alert update topic: {settings.KAFKA_ALERT_UPDATE_TOPIC}")
        data_consumer.subscribe(
            [table_change.get_topic_display() for table_change in table_changes]
        )
        debug(
            f"Subscribed to topics: {[table_change.get_topic_display() for table_change in table_changes]}"
        )

    @classmethod
    def _execute_tasks(
        cls, *, data_consumer: Consumer, alert_update_consumer: Consumer
    ) -> None:
        """Start executing tasks based on the received Kafka messages.

        Args:
            data_consumer (Consumer): Consumer for data messages.
            alert_update_consumer (Consumer): Consumer for alert update messages.

        Returns:
            None: This function does not return anything.
        """
        futures = set()
        with concurrent.futures.ThreadPoolExecutor(
            max_workers=settings.KAFKA_THREAD_POOL_SIZE,
            thread_name_prefix="kafka_listener",
        ) as executor:
            try:
                while cls.running:
                    if len(futures) < settings.KAFKA_MAX_CONCURRENT_JOBS:
                        cls._handle_messages(
                            data_consumer=data_consumer,
                            alert_update_consumer=alert_update_consumer,
                            futures=futures,
                            executor=executor,
                        )

                    cls._wait_for_futures_to_complete(futures=futures)

            except Exception as e:
                cls._handle_exception(
                    e=e,
                    data_consumer=data_consumer,
                    alert_update_consumer=alert_update_consumer,
                )

    @classmethod
    def _handle_messages(
        cls,
        *,
        data_consumer: Consumer,
        alert_update_consumer: Consumer,
        futures: set[concurrent.futures.Future],
        executor: concurrent.futures.ThreadPoolExecutor,
    ) -> None:
        """Handle alert and data messages from Kafka.

        Args:
            data_consumer (KafkaConsumer): Consumer for data messages.
            alert_update_consumer (KafkaConsumer): Consumer for alert update messages.
            futures (set[Future]): Set of futures for tracking ongoing tasks.
            executor (ThreadPoolExecutor): Executor for running tasks.

        Returns:
            None: This function does not return anything.
        """
        cls._handle_alert_message(
            alert_update_consumer=alert_update_consumer, data_consumer=data_consumer
        )
        cls._handle_data_messages(
            data_consumer=data_consumer, futures=futures, executor=executor
        )

    @classmethod
    def _handle_alert_message(
        cls, *, alert_update_consumer: Consumer, data_consumer: Consumer
    ) -> None:
        """Handle alert messages from Kafka and update subscriptions if necessary.

        Args:
            alert_update_consumer (Consumer): Consumer for alert update messages.
            data_consumer (Consumer): Consumer for data messages.

        Returns:
            None: This function does not return anything.
        """
        alert_message = cls._get_message(
            consumer=alert_update_consumer, timeout=POLL_TIMEOUT
        )

        if alert_message is not None:
            debug(f"Received alert update: {alert_message.value()}")
            cls._update_subscriptions(
                data_consumer=data_consumer, table_changes=cls._get_topic_list()
            )

    @classmethod
    def _handle_data_messages(
        cls,
        *,
        data_consumer: Consumer,
        futures: set[concurrent.futures.Future],
        executor: concurrent.futures.ThreadPoolExecutor,
    ) -> None:
        """Handle data messages from Kafka and submit tasks for processing.

        Args:
            data_consumer (KafkaConsumer): Consumer for data messages.
            futures (set[Future]): Set of futures for tracking ongoing tasks.
            executor (ThreadPoolExecutor): Executor for running tasks.

        Returns:
            None: This function does not return anything.
        """
        data_messages = cls._get_messages(consumer=data_consumer, timeout=POLL_TIMEOUT)

        for data_message in data_messages:
            cls._process_data_message(
                data_message=data_message, futures=futures, executor=executor
            )

    @classmethod
    def _process_data_message(
        cls,
        *,
        data_message: Message,
        futures: set[concurrent.futures.Future],
        executor: concurrent.futures.ThreadPoolExecutor,
    ) -> None:
        """Process a single data message.

        Args:
            data_message (str): The data message to process.
            futures (set[Future]): Set of futures for tracking ongoing tasks.
            executor (ThreadPoolExecutor): Executor for running tasks.

        Returns:
            None: This function does not return anything.
        """
        if (
            data_message is not None
            and not data_message.error()
            and data_message.value() is not None
        ):
            debug(
                f"Received data: {data_message.value().decode('utf-8')} from topic: {data_message.topic()}"
            )

            if associated_table_change := next(
                (
                    table_change
                    for table_change in cls._get_topic_list()
                    if table_change.get_topic_display() == data_message.topic()
                ),
                None,
            ):
                debug(f"Table Change Alert found. {associated_table_change.name}")
                future = executor.submit(
                    cls._process_message,
                    data_message,
                    associated_table_change,
                )
                futures.add(future)

    @classmethod
    def _wait_for_futures_to_complete(
        cls, *, futures: set[concurrent.futures.Future]
    ) -> None:
        """Wait for tasks to complete, and if they raise any exceptions, log the error.

        Args:
            futures (set[Future]): Set of futures for tracking ongoing tasks.

        Returns:
            None: This function does not return anything.
        """
        if len(futures) >= settings.KAFKA_MAX_CONCURRENT_JOBS:
            done, futures = concurrent.futures.wait(
                futures, return_when=concurrent.futures.FIRST_COMPLETED
            )

        done, futures = concurrent.futures.wait(
            futures, return_when=concurrent.futures.FIRST_COMPLETED
        )

        for future in done:
            try:
                future.result()
            except Exception as e:
                error(f"Error processing message: {e}", exc_info=True)

        futures = {f for f in futures if not f.done()}

    @classmethod
    def _handle_exception(
        cls,
        *,
        e: Exception,
        data_consumer: Consumer,
        alert_update_consumer: Consumer,
    ) -> None:
        """Handle exceptions that occur while processing tasks. If the exception is due to all Kafka brokers being down,
        it attempts to reconnect.

        Args:
            e (Exception): The exception to handle.
            data_consumer (Consumer): Consumer for data messages.
            alert_update_consumer (Consumer): Consumer for alert update messages.

        Returns:
            None: This function does not return anything.
        """
        if (
            isinstance(e, KafkaException)
            and e.args[0].code() == KafkaError._ALL_BROKERS_DOWN
        ):
            error("All brokers are down. Attempting to reconnect...", exc_info=True)
            data_consumer, alert_update_consumer = cls._connect()
        else:
            error("An unexpected error occurred: ", exc_info=True)
            raise e
