"""
COPYRIGHT 2022 MONTA

This file is part of Monta.

Monta is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

Monta is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with Monta.  If not, see <https://www.gnu.org/licenses/>.
"""
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

        IntervalSchedule.objects.bulk_create(minute_objs)
        IntervalSchedule.objects.bulk_create(hour_objs)
        IntervalSchedule.objects.bulk_create(day_objs)

        self.stdout.write(self.style.SUCCESS("Celery beat configurations created successfully."))