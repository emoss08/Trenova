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

from .models import EquipmentType, EquipmentTypeDetail


class EquipmentTypeDetailAdmin(admin.StackedInline):
    """
    Equipment Type Detail Admin
    """
    model: Type[EquipmentTypeDetail] = EquipmentTypeDetail
    can_delete: bool = False
    verbose_name_plural: str = "Equipment Type Details"
    fk_name: str = "equipment_type"
    extra: int = 0
    autocomplete_fields: tuple[str, ...] = ("equipment_type", "organization")


@admin.register(EquipmentType)
class EquipmentTypeAdmin(admin.ModelAdmin):
    """
    Equipment Type Admin
    """
    model: Type[EquipmentType] = EquipmentType
    list_display: tuple[str, ...] = ("name", "description")
    search_fields: tuple[str, ...] = ("name", "description")
    inlines: tuple[Type[EquipmentTypeDetailAdmin], ...] = (EquipmentTypeDetailAdmin,)
    autocomplete_fields: tuple[str, ...] = ("organization",)
