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

import os
import socket
from pathlib import Path
from typing import Any, TypeAlias

from confluent_kafka import KafkaException, admin
from environ import environ
from rich import print as rprint

env = environ.Env()
ENV_DIR = Path(__file__).parent.parent
environ.Env.read_env(os.path.join(ENV_DIR, ".env"))

ConsumerGroupMetadata: TypeAlias = dict[str, list[dict[str, Any]]]


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
        self.kafka_host = env("KAFKA_HOST")
        self.kafka_port = env("KAFKA_PORT")
        self.admin_client = admin.AdminClient(
            {"bootstrap.servers": env("KAFKA_BOOTSTRAP_SERVERS")}
        )

    def __str__(self) -> str:
        """Returns the string representation of the KafkaManager instance.

        Returns:
            str: The string representation of the KafkaManager instance.
        """

        return f"KafkaManager(bootstrap_servers={env('KAFKA_BOOTSTRAP_SERVERS')})"

    def is_kafka_available(self, *, timeout: int = 5) -> bool:
        """Checks if the Kafka server is available.

        This method tries to create a socket connection to the Kafka server with the given host and port.
        If the connection is successful, the Kafka server is considered available.

        Args:
            timeout (int, optional): The maximum time to wait for a connection. Default is 5 seconds.

        Returns:
            bool: True if the Kafka server is available, False otherwise.
        """

        try:
            sock: socket = socket.create_connection(
                (self.kafka_host, self.kafka_port), timeout=timeout
            )
            sock.close()
            return True
        except OSError as err:
            rprint(f"[red]Kafka is not available: {err}[/]")
            return False

    def get_available_topics(self) -> list[tuple]:
        """Fetches the list of available topics from the Kafka server.

        If the ``admin_client`` is not available or the Kafka server is not available,
        this method returns an empty list. Otherwise, it fetches the metadata from the
        Kafka server, extracts the topic names, and returns them as a list of tuples
        for use in Django choices.

        Returns:
            list[tuple]: A list of tuples with available topics from the Kafka server. Each tuple has two elements: the topic name and the topic name again.
        """

        if self.admin_client is None:
            # raise KafkaException("Kafka admin client is not available.")
            return []
        if not self.is_kafka_available():
            # raise KafkaException("Kafka is not available.")
            return []

        try:
            topic_metadata = self.admin_client.list_topics(timeout=5)
            return [
                (topic, topic)
                for topic in list(topic_metadata.topics.keys())
                if not topic.startswith("__")
            ]
        except KafkaException as ke:
            rprint(f"[red]Failed to fetch topics from Kafka: {ke}[/]")
            return []

    def create_topic(
        self, *, topic: str, num_partitions: int, replication_factor: int
    ) -> None:
        """Creates a new Kafka topic.

        Args:
            topic (str): The name of the topic to be created.
            num_partitions (int): The number of partitions for the new topic.
            replication_factor (int): The replication factor for the new topic.

        Returns:
            None: This function does not return anything.
        """

        new_topic = admin.NewTopic(topic, num_partitions, replication_factor)
        self.admin_client.create_topics([new_topic])

    def delete_topic(self, *, topic: str) -> None:
        """Deletes the specified Kafka topic.

        Args:
            topic (str): The name of the topic to be deleted.

        Returns:
            None: This function does not return anything.
        """

        self.admin_client.delete_topics([topic])

    def increase_topic_partitions(self, *, topic: str, new_partitions: int) -> None:
        """Increases the number of partitions for the specified Kafka topic.

        Args:
            topic (str): The name of the topic.
            new_partitions (int): The new total number of partitions for the topic.

        Returns:
            None: This function does not return anything.
        """

        new_partitions = admin.NewPartitions(topic, new_partitions)
        self.admin_client.create_partitions([new_partitions])

    def list_consumer_groups(self) -> list[str]:
        """Lists all consumer group IDs.

        Returns:
            list[str]: A list of all consumer group IDs.
        """

        return list(self.admin_client.list_groups().groups.keys())

    def describe_topic(self, *, topic_name: str) -> dict[str, str]:
        """Describe a specific topic's configuration.

        Args:
            topic_name (str): The name of the topic.

        Returns:
            dict[str, str]: A dictionary representing the configuration of the topic.
        """

        resource = admin.ConfigResource(admin.ResourceType.TOPIC, topic_name)
        config = self.admin_client.describe_configs([resource])
        return {
            config_entry[0]: config_entry[1].value
            for config_entry in config[topic_name].items()
        }

    def alter_topic_config(
        self, *, topic_name: str, config_dict: dict[str, str]
    ) -> None:
        """Alter the configuration of a topic.

        Args:
            topic_name (str): The name of the topic.
            config_dict (Dict[str, str]): A dictionary representing the new configuration of the topic.

        Returns:
            None: This function does not return anything.
        """

        resource = admin.ConfigResource(admin.ResourceType.TOPIC, topic_name)
        entries = {
            key: admin.NewPartitions(value) for key, value in config_dict.items()
        }
        self.admin_client.alter_configs({resource: entries})

    def describe_consumer_group(self, *, group_id: str) -> ConsumerGroupMetadata:
        """Describe a specific consumer group.

        Args:
            group_id (str): The id of the consumer group.

        Returns:
            dict[str, str]: A dictionary representing the configuration of the consumer group.
        """

        group_description = self.admin_client.list_groups([group_id]).groups[group_id]
        return {
            "state": group_description.state,
            "protocol_type": group_description.protocol_type,
            "protocol": group_description.protocol,
            "members": [
                {
                    "id": member.id,
                    "client_id": member.client_id,
                    "client_host": member.client_host,
                    "member_metadata": member.member_metadata,
                    "member_assignment": member.member_assignment,
                }
                for member in group_description.members
            ],
        }
