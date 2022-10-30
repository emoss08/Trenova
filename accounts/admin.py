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

from accounts import models


class ProfileInline(admin.StackedInline):
    """
    Profile inline
    """

    model: Type[models.Profile] = models.Profile
    can_delete: bool = False
    verbose_name_plural: str = "profiles"
    fk_name: str = "user"
    extra: int = 0


@admin.register(models.User)
class UserAdmin(admin.ModelAdmin):
    """
    User Admin
    """

    inlines: tuple[Type[ProfileInline]] = (ProfileInline,)
    list_display: tuple[str, ...] = (
        "username",
        "email",
    )


@admin.register(models.JobTitle)
class JobTitleAdmin(admin.ModelAdmin):
    """
    Job title admin
    """

    fieldsets = (
        (None, {"fields": ("organization", "name", "is_active", "description")}),
    )
