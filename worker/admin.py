# -*- coding: utf-8 -*-
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

from typing import Type

from django.contrib import admin

from .models import Worker, WorkerComment, WorkerContact, WorkerProfile


class WorkerProfileAdmin(admin.StackedInline):
    """
    Worker Profile Admin
    """

    model: Type[WorkerProfile] = WorkerProfile
    can_delete: bool = False
    verbose_name_plural: str = "Worker Profile"
    fk_name: str = "worker"
    extra: int = 0
    autocomplete_fields: tuple[str, ...] = ("worker", "organization")


class WorkerContactAdmin(admin.StackedInline):
    """
    Worker Contact Admin
    """

    model: Type[WorkerContact] = WorkerContact
    verbose_name_plural: str = "Worker Contact"
    fk_name: str = "worker"
    extra: int = 0
    autocomplete_fields: tuple[str, ...] = ("worker", "organization")


class WorkerCommentAdmin(admin.StackedInline):
    """
    Worker Comment Admin
    """

    model: Type[WorkerComment] = WorkerComment
    verbose_name_plural: str = "Worker Comment"
    fk_name: str = "worker"
    extra: int = 0
    autocomplete_fields: tuple[str, ...] = ("worker", "organization")


@admin.register(Worker)
class WorkerAdmin(admin.ModelAdmin[Worker]):
    """
    Worker Admin
    """

    model: Type[Worker] = Worker
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
    autocomplete_fields: tuple[str, ...] = ("organization",)
    fieldsets = (
        (
            None,
            {
                "fields": (
                    "organization",
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
