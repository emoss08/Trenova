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
from nested_inline.admin import (
    NestedModelAdmin,
    NestedStackedInline,
    NestedTabularInline,
)

from movements.models import Movement
from order import models
from stops.models import Stop
from utils.admin import GenericAdmin, GenericStackedInline


class OrderDocumentationInline(
    NestedTabularInline[models.OrderDocumentation, models.Order]
):
    """
    Order documentation inline
    """

    model: type[models.OrderDocumentation] = models.OrderDocumentation


class OrderCommentInline(GenericStackedInline[models.OrderComment, models.Order]):
    """
    Order comment inline
    """

    model: type[models.OrderComment] = models.OrderComment


class AdditionalChargeInline(
    GenericStackedInline[models.AdditionalCharge, models.Order]
):
    """
    Order Additional Charge inline
    """

    model: type[models.AdditionalCharge] = models.AdditionalCharge


class StopInline(NestedStackedInline):
    """
    Order Stop Inline
    """

    model = Stop
    extra = 1


class MovementInline(NestedStackedInline):
    """
    Order Movement Inline
    """

    model: type[Movement] = Movement
    inlines = [StopInline]


@admin.register(models.OrderType)
class OrderTypeAdmin(GenericAdmin[models.OrderType]):
    """
    Order Type Admin
    """

    list_display = (
        "name",
        "description",
    )
    search_fields = ("name", "description")


@admin.register(models.ReasonCode)
class ReasonCodeAdmin(GenericAdmin[models.ReasonCode]):
    """
    Reason Code Admin
    """

    list_display = (
        "code",
        "description",
    )
    search_fields = ("code", "description")


@admin.register(models.OrderControl)
class OrderControlAdmin(GenericAdmin[models.OrderControl]):
    """
    Order Control Admin
    """

    list_display = (
        "organization",
        "auto_rate_orders",
    )
    search_fields = ("organization", "auto_rate_orders")


@admin.register(models.Order)
class OrderAdmin(NestedModelAdmin):
    """
    Order Admin
    """

    list_display = (
        "pro_number",
        "status",
        "origin_location",
        "destination_location",
    )
    search_fields = ("pro_number",)
    fieldsets = (
        (
            None,
            {
                "fields": (
                    "organization",
                    "status",
                    "order_type",
                    "revenue_code",
                    "entered_by",
                )
            },
        ),
        (
            "Order Information",
            {
                "fields": (
                    "origin_location",
                    "origin_address",
                    "origin_appointment",
                    "destination_location",
                    "destination_address",
                    "destination_appointment",
                )
            },
        ),
        (
            "Billing Details",
            {
                "fields": (
                    "mileage",
                    "other_charge_amount",
                    "freight_charge_amount",
                    "sub_total",
                    "rate_method",
                    "customer",
                    "pieces",
                    "weight",
                    "ready_to_bill",
                    "bill_date",
                    "billed",
                    "transferred_to_billing",
                    "billing_transfer_date",
                ),
            },
        ),
        (
            "Dispatch Details",
            {
                "fields": (
                    "equipment_type",
                    "commodity",
                    "hazmat",
                    "temperature_min",
                    "temperature_max",
                    "bol_number",
                    "consignee_ref_number",
                    "comment",
                )
            },
        ),
    )
    inlines = (
        MovementInline,
        OrderDocumentationInline,
    )
