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

from core.mixins import MontaAdminMixin, MontaStackedInlineMixin
from .models import Worker, WorkerComment, WorkerContact, WorkerProfile


class WorkerProfileAdmin(MontaStackedInlineMixin[Worker, WorkerProfile]):
    """
    Worker Profile Admin
    """

    model: type[WorkerProfile] = WorkerProfile
    can_delete: bool = False
    verbose_name_plural: str = "Worker Profile"
    fk_name: str = "worker"
    extra: int = 0


class WorkerContactAdmin(MontaStackedInlineMixin[Worker, WorkerContact]):
    """
    Worker Contact Admin
    """

    model: type[WorkerContact] = WorkerContact
    verbose_name_plural: str = "Worker Contact"
    fk_name: str = "worker"
    extra: int = 0


class WorkerCommentAdmin(MontaStackedInlineMixin[Worker, WorkerComment]):
    """
    Worker Comment Admin
    """

    model: type[WorkerComment] = WorkerComment
    verbose_name_plural: str = "Worker Comment"
    fk_name: str = "worker"
    extra: int = 0


@admin.register(Worker)
class WorkerAdmin(MontaAdminMixin[Worker]):
    """
    Worker Admin
    """

    model: type[Worker] = Worker
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
