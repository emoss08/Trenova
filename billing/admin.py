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

from billing.models import (
    AccessorialCharge,
    BillingControl,
    ChargeType,
    DocumentClassification,
)
from utils.admin import GenericAdmin


@admin.register(BillingControl)
class BillingControlAdmin(GenericAdmin[BillingControl]):
    """
    Billing Control Admin
    """

    model: type[BillingControl] = BillingControl
    list_display = ("organization", "auto_bill_orders")
    search_fields = ("organization", "auto_bill_orders")


@admin.register(DocumentClassification)
class DocumentClassificationAdmin(GenericAdmin[DocumentClassification]):
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
class ChargeTypeAdmin(GenericAdmin[ChargeType]):
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
class AccessorialChargeAdmin(GenericAdmin[AccessorialCharge]):
    """
    Accessorial Charge Admin
    """

    model: type[AccessorialCharge] = AccessorialCharge
    list_display = (
        "code",
        "charge_amount",
        "method",
    )
    search_fields = ("code",)
