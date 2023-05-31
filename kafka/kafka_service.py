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

from confluent_kafka import Consumer, KafkaException
from environ import environ
from pathlib import Path
import os
import socket
from rich import print as rprint

# Load environment variables
env = environ.Env()
ENV_DIR = Path(__file__).parent.parent
environ.Env.read_env(os.path.join(ENV_DIR, ".env"))

# Kafka configuration
kafka_bootstrap_servers = env("KAFKA_BOOTSTRAP_SERVERS").split(":")
kafka_host = kafka_bootstrap_servers[0]
kafka_port = int(kafka_bootstrap_servers[1])


class KafkaManager:
    _instance: KafkaManager | None = None
    __initialized: bool

    def __new__(cls) -> "KafkaManager":
        if cls._instance is None:
            cls._instance = super(KafkaManager, cls).__new__(cls)
            cls._instance.__initialized = False
        return cls._instance

    def __init__(self):
        if self.__initialized:
            return
        self.__initialized = True

        self.consumer_conf = {
            "bootstrap.servers": env("KAFKA_BOOTSTRAP_SERVERS"),
            "group.id": env("KAFKA_GROUP_ID"),
            "auto.offset.reset": "earliest",
        }
        self.kafka_bootstrap_servers = env("KAFKA_BOOTSTRAP_SERVERS").split(":")
        self.kafka_host = kafka_bootstrap_servers[0]
        self.kafka_port = int(kafka_bootstrap_servers[1])
        self.consumer = None

    @staticmethod
    def is_kafka_available(*, host: str, port: int, timeout: int = 5) -> bool:
        try:
            sock = socket.create_connection((host, port), timeout=timeout)
            sock.close()
            return True
        except socket.error as err:
            rprint(f"[red]Kafka is not available: {err}[/]")
            return False

    def create_open_consumer(self) -> Consumer:
        try:
            self.consumer = Consumer(self.consumer_conf)
        except KafkaException as ke:
            rprint(f"[red]Failed to create Kafka consumer: {ke}[/]")
            self.consumer = None

    def get_available_topics(self) -> list[tuple]:
        if self.consumer is None:
            return []

        if not self.is_kafka_available(host=self.kafka_host, port=self.kafka_port):
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

    def close_consumer(self) -> None:
        if self.consumer is not None:
            self.consumer.close()

    def get_topics(self) -> list[tuple] | list:
        try:
            # Create consumer
            self.create_open_consumer()
            # Get available topics
            topics = self.get_available_topics()
            # Close consumer after fetching metadata
            self.close_consumer()
            return topics
        except KafkaException as ke:
            rprint(f"[red]Failed to fetch topics from Kafka: {ke}[/]")
            return []
