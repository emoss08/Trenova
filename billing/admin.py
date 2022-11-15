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

from core.mixins import MontaAdminMixin
from .models import AccessorialCharge, ChargeType, DocumentClassification


@admin.register(DocumentClassification)
class DocumentClassificationAdmin(MontaAdminMixin[DocumentClassification]):
    """
    Document Classification Admin
    """

    model: type[DocumentClassification] = DocumentClassification
    list_display = (
        "name",
        "description",
    )
    search_fields = ("name",)


@admin.register(ChargeType)
class ChargeTypeAdmin(MontaAdminMixin[ChargeType]):
    """
    Charge Type Admin
    """

    model: type[ChargeType] = ChargeType
    list_display = (
        "name",
        "description",
    )
    search_fields = ("name",)


@admin.register(AccessorialCharge)
class AccessorialChargeAdmin(MontaAdminMixin[AccessorialCharge]):
    """
    Accessorial Charge Admin
    """

    model: type[AccessorialCharge] = AccessorialCharge
    list_display = (
        "code",
        "is_fuel_surcharge",
    )
    search_fields = ("code",)
