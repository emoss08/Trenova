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

import contextlib
import os
import socket
from pathlib import Path

from confluent_kafka import Consumer, KafkaException
from environ import environ
from rich import print as rprint

# Load environment variables
env = environ.Env()
ENV_DIR = Path(__file__).parent.parent
environ.Env.read_env(os.path.join(ENV_DIR, ".env"))


class KafkaManager:
    """Manages the Kafka connection and related operations.

    This class serves as a Singleton manager for Kafka related operations. This includes
    creating a Kafka consumer, checking Kafka server availability, getting available topics,
    and closing the Kafka consumer.

    Attributes:
        _instance (KafkaManager | None): The single instance of KafkaManager, None initially.
        __initialized (bool): A flag indicating whether the KafkaManager instance is initialized.
    """

    _instance: KafkaManager | None = None
    __initialized: bool

    def __new__(cls) -> KafkaManager:
        """Creates a new instance of KafkaManager if it doesn't exist already.

        Overrides the __new__ method to make KafkaManager a Singleton.

        Returns:
            KafkaManager: The single instance of KafkaManager.
        """
        if cls._instance is None:
            cls._instance = super().__new__(cls)
            cls._instance.__initialized = False
        return cls._instance

    def __init__(self):
        """Initializes the KafkaManager instance with consumer configuration.

        Only performs initialization the first time this instance is created.
        """
        if self.__initialized:
            return
        self.__initialized = True

        self.consumer_conf = {
            "bootstrap.servers": env("KAFKA_BOOTSTRAP_SERVERS"),
            "group.id": env("KAFKA_GROUP_ID"),
            "auto.offset.reset": "latest",
        }
        self.kafka_host = env("KAFKA_HOST")
        self.kafka_port = env("KAFKA_PORT")
        self.consumer = None

    def __str__(self) -> str:
        """Returns the string representation of the KafkaManager instance.

        Returns:
            str: The string representation of the KafkaManager instance.
        """
        return f"KafkaManager(bootstrap_servers={self.consumer_conf['bootstrap.servers']}, group_id={self.consumer_conf['group.id']})"

    def __del__(self) -> None:
        """Destructor for the KafkaManager class.

        Returns:
            None: This function does not return anything.
        """
        with contextlib.suppress(Exception):
            self._close_consumer()

    @staticmethod
    def _is_kafka_available(*, host: str, port: int, timeout: int = 5) -> bool:
        """Checks if the Kafka server is available.

        This method tries to create a socket connection to the Kafka server with the given host and port.
        If the connection is successful, the Kafka server is considered available.

        Args:
            host (str): The hostname of the Kafka server.
            port (int): The port number of the Kafka server.
            timeout (int, optional): The maximum time to wait for a connection. Default is 5 seconds.

        Returns:
            bool: True if the Kafka server is available, False otherwise.
        """
        try:
            sock = socket.create_connection((host, port), timeout=timeout)
            sock.close()
            return True
        except OSError as err:
            rprint(f"[red]Kafka is not available: {err}[/]")
            return False

    def _create_open_consumer(self) -> Consumer:
        """Creates and opens a Kafka consumer.

        This method tries to create a Kafka consumer with the consumer configuration provided
        during initialization. If successful, the consumer is stored in the instance variable `self.consumer`.

        Returns:
            Consumer: The Kafka consumer.
        """
        try:
            self.consumer = Consumer(self.consumer_conf)
        except KafkaException as ke:
            rprint(f"[red]Failed to create Kafka consumer: {ke}[/]")
            self.consumer = None

    def _get_available_topics(self) -> list[tuple]:
        """Fetches the list of available topics from the Kafka server.

        If the consumer is not available or the Kafka server is not available,
        this method returns an empty list. Otherwise, it fetches the metadata from the
        Kafka server, extracts the topic names, and returns them as a list of tuples
        for use in Django choices.

        Returns:
            list[tuple]: A list of tuples with available topics from the Kafka server. Each tuple has two elements: the topic name and the topic name again.
        """
        if self.consumer is None:
            return []

        if not self._is_kafka_available(host=self.kafka_host, port=self.kafka_port):
            return []

        try:
            # set timeout for metadata fetch, e.g., 5 seconds
            cluster_metadata = self.consumer.list_topics()

            topics = cluster_metadata.topics

            # Create 2-tuples for Django choices
            return [(topic, topic) for topic in topics.keys()]
        except KafkaException as ke:
            rprint(f"[red]Failed to fetch topics from Kafka: {ke}[/]")
            return []

    def _close_consumer(self) -> None:
        """Closes the Kafka consumer.

        If a consumer has been created and opened, this method closes the consumer.

        Returns:
            None: This function does not return anything.
        """
        if self.consumer is not None:
            self.consumer.close()

    def get_topics(self) -> list[tuple] | list:
        """Creates a Kafka consumer, fetches available topics, and then closes the consumer.

        This method handles the overall process of fetching available topics from the Kafka server.
        If any step fails, it returns an empty list.

        Returns:
            list[tuple] | list: A list of tuples with available topics from the Kafka server, or an empty list in case of failure.
        """
        try:
            # Create consumer
            self._create_open_consumer()

            # Get available topics
            topics = self._get_available_topics()

            # Close consumer after fetching metadata
            self._close_consumer()

            return topics
        except KafkaException as ke:
            rprint(f"[red]Failed to fetch topics from Kafka: {ke}[/]")
            return []
