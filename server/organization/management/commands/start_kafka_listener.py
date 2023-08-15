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
from multiprocessing import Process
from typing import Any

from django.core.management import BaseCommand

from kafka import services


class Command(BaseCommand):
    help = "Starts the KafkaListener"

    def handle(self, *args: Any, **options: Any) -> None:
        # Close the existing database connections
        listener = services.KafkaListener()
        listener._close_old_connections()
        listener.listen()
        p = Process(target=listener.listen)
        p.start()

        # Save the PID to a file
        with open("kafka_listener_pid.txt", "w") as f:
            f.write(str(p.pid))
