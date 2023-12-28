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

from django.contrib import admin

from utils.admin import GenericAdmin, GenericStackedInline
from worker import models


class WorkerProfileAdmin(GenericStackedInline[models.Worker, models.WorkerProfile]):
    """
    Worker Profile Admin
    """

    model = models.WorkerProfile
    can_delete = False
    verbose_name_plural = "Worker Profile"
    fk_name = "worker"


class WorkerContactAdmin(GenericStackedInline[models.Worker, models.WorkerContact]):
    """
    Worker Contact Admin
    """

    model = models.WorkerContact
    verbose_name_plural = "Worker Contact"
    fk_name = "worker"


class WorkerCommentAdmin(GenericStackedInline[models.Worker, models.WorkerComment]):
    """
    Worker Comment Admin
    """

    model = models.WorkerComment
    verbose_name_plural = "Worker Comment"
    fk_name = "worker"


@admin.register(models.Worker)
class WorkerAdmin(GenericAdmin[models.Worker]):
    """
    Worker Admin
    """

    model = models.Worker
    list_display = (
        "code",
        "is_active",
        "worker_type",
        "first_name",
        "last_name",
    )
    search_fields = (
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
                    "fleet_code",
                    "entered_by",
                    "profile_picture",
                    "thumbnail",
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


@admin.register(models.WorkerHOS)
class WorkerHOSAdmin(GenericAdmin[models.WorkerHOS]):
    """
    Worker HOS Admin
    """

    model = models.WorkerHOS
    list_display = ("worker", "drive_time", "off_duty_time")
    search_fields = ("worker", "drive_time", "off_duty_time")
