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

from typing import Any

from django.core.management import BaseCommand
from django.db.transaction import atomic
from django_celery_beat.models import IntervalSchedule


class Command(BaseCommand):
    """
    Django command to create system user.
    """

    help = "Command for creating initial celery beat configurations."

    @atomic(using="default")
    def handle(self, *args: Any, **options: Any) -> None:
        """
        Handle the command.

        Args:
            *args (Any): Additional arguments passed to the command
            **options (Any): Additional options passed to the command

        Returns:
            None: None
        """
        self.stdout.write(
            self.style.HTTP_INFO("Creating celery beat configurations...")
        )

        self.stdout.write(
            self.style.HTTP_INFO(
                "Checking if celery beat configurations already exist..."
            )
        )
        if IntervalSchedule.objects.all().exists():
            self.stdout.write(
                self.style.NOTICE("Celery beat configurations already exist.")
            )
            return

        self.stdout.write(
            self.style.NOTICE("Celery beat configurations do not exist. Creating...")
        )

        micro_second_objs = [
            IntervalSchedule(every=micro_second, period=IntervalSchedule.MICROSECONDS)
            for micro_second in range(1, 1000)
        ]
        minute_objs = [
            IntervalSchedule(every=minute, period=IntervalSchedule.MINUTES)
            for minute in range(1, 60)
        ]
        hour_objs = [
            IntervalSchedule(every=hour, period=IntervalSchedule.HOURS)
            for hour in range(1, 24)
        ]
        day_objs = [
            IntervalSchedule(every=day, period=IntervalSchedule.DAYS)
            for day in range(1, 7)
        ]

        IntervalSchedule.objects.bulk_create(micro_second_objs)
        IntervalSchedule.objects.bulk_create(minute_objs)
        IntervalSchedule.objects.bulk_create(hour_objs)
        IntervalSchedule.objects.bulk_create(day_objs)

        self.stdout.write(
            self.style.SUCCESS("Celery beat configurations created successfully.")
        )
