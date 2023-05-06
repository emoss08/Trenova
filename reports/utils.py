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


from django_celery_beat.models import CrontabSchedule

from reports import exceptions, models


def get_crontab_schedule(
    *, schedule_type: str, instance: models.ScheduledReport
) -> tuple[CrontabSchedule, str]:
    """Get or create a CrontabSchedule object based on the schedule type and scheduled report instance.

    Args:
        schedule_type (models.ScheduleType): The schedule type (DAILY, WEEKLY, MONTHLY) of the scheduled report.
        instance (models.ScheduledReport): The scheduled report instance.

    Returns:
        Tuple[CrontabSchedule, str]: A tuple containing the CrontabSchedule object and the task type ("crontab").

    Raises:
        exceptions.InvalidScheduleTypeException: If the schedule type is not valid.
    """
    if schedule_type == models.ScheduleType.DAILY:
        schedule_filters = {
            "hour": instance.time.hour,
            "minute": instance.time.minute,
            "timezone": instance.timezone,
        }
    elif schedule_type == models.ScheduleType.WEEKLY:
        weekdays = ",".join([str(weekday.id) for weekday in instance.day_of_week.all()])
        schedule_filters = {
            "day_of_week": weekdays,
            "hour": instance.time.hour,
            "minute": instance.time.minute,
            "timezone": instance.timezone,
        }
    elif schedule_type == models.ScheduleType.MONTHLY:
        schedule_filters = {
            "day_of_month": instance.day_of_month,
            "hour": instance.time.hour,
            "minute": instance.time.minute,
            "timezone": instance.timezone,
        }
    else:
        raise exceptions.InvalidScheduleTypeException("Invalid schedule type.")

    schedule, created = CrontabSchedule.objects.get_or_create(**schedule_filters)
    return schedule, "crontab"
