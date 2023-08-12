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

from django.contrib import admin
from utils.admin import GenericAdmin, GenericStackedInline
from worker import models


class WorkerProfileAdmin(GenericStackedInline[models.Worker, models.WorkerProfile]):
    """
    Worker Profile Admin
    """

    model: type[models.WorkerProfile] = models.WorkerProfile
    can_delete = False
    verbose_name_plural = "Worker Profile"
    fk_name = "worker"


class WorkerContactAdmin(GenericStackedInline[models.Worker, models.WorkerContact]):
    """
    Worker Contact Admin
    """

    model: type[models.WorkerContact] = models.WorkerContact
    verbose_name_plural = "Worker Contact"
    fk_name = "worker"


class WorkerCommentAdmin(GenericStackedInline[models.Worker, models.WorkerComment]):
    """
    Worker Comment Admin
    """

    model: type[models.WorkerComment] = models.WorkerComment
    verbose_name_plural = "Worker Comment"
    fk_name = "worker"


@admin.register(models.Worker)
class WorkerAdmin(GenericAdmin[models.Worker]):
    """
    Worker Admin
    """

    model: type[models.Worker] = models.Worker
    list_display: tuple[str, ...] = (
        "code",
        "is_active",
        "worker_type",
        "first_name",
        "last_name",
    )
    search_fields: tuple[str, ...] = (
        "code",
        "worker_type",
        "first_name",
        "last_name",
    )
    fieldsets = (
        (
            None,
            {
                "fields": (
                    "is_active",
                    "worker_type",
                    "depot",
                    "manager",
                    "fleet",
                    "entered_by",
                )
            },
        ),
        (
            "Personal Information",
            {
                "fields": (
                    "first_name",
                    "last_name",
                    "address_line_1",
                    "address_line_2",
                    "city",
                    "state",
                    "zip_code",
                ),
            },
        ),
    )
    inlines = (
        WorkerProfileAdmin,
        WorkerContactAdmin,
        WorkerCommentAdmin,
    )


@admin.register(models.WorkerTimeAway)
class WorkerTimeAwayAdmin(GenericAdmin[models.WorkerTimeAway]):
    """
    Worker Time Away Admin
    """

    model = models.WorkerTimeAway
    list_display = ("worker", "start_date", "end_date")
    search_fields = ("worker", "start_date", "end_date")
