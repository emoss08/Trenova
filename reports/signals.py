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
import json
from typing import Any

from django_celery_beat.models import CrontabSchedule, PeriodicTask

from reports import models
from reports.models import ScheduleType


def create_scheduled_task(sender: models.ScheduledReport, instance: models.ScheduledReport, created: bool, **kwargs: Any) -> None:
    # Create or update the schedule based on schedule_type
    if instance.schedule_type == ScheduleType.DAILY:
        schedule, _ = CrontabSchedule.objects.get_or_create(
            hour=instance.time.hour,
            minute=instance.time.minute,
        )
        task_type = 'crontab'
    elif instance.schedule_type == ScheduleType.WEEKLY:
        schedule, _ = CrontabSchedule.objects.get_or_create(
            day_of_week=instance.day_of_week,
            hour=instance.time.hour,
            minute=instance.time.minute,
        )
        task_type = 'crontab'
    elif instance.schedule_type == ScheduleType.MONTHLY:
        schedule, _ = CrontabSchedule.objects.get_or_create(
            day_of_month=instance.day_of_month,
            hour=instance.time.hour,
            minute=instance.time.minute,
        )
        task_type = 'crontab'
    else:
        raise ValueError("Invalid schedule_type")

    # Create or update the periodic task
    task, created_task = PeriodicTask.objects.get_or_create(
        crontab=schedule if task_type == 'crontab' else None,
        interval=schedule if task_type == 'interval' else None,
        name=f"Send scheduled report {instance.pk}",
        task='reports.tasks.send_scheduled_report',
        args=json.dumps([str(instance.pk)]),
    )

    if not created_task:
        setattr(task, task_type, schedule)
        task.args = json.dumps([str(instance.pk)])
        task.save()