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
from typing import Type

from django.contrib import admin

from order import models
from utils.admin import GenericAdmin, GenericStackedInline


class OrderDocumentationInline(
    GenericStackedInline[models.OrderDocumentation, models.Order]
):
    """
    Order documentation inline
    """

    model: Type[models.OrderDocumentation] = models.OrderDocumentation


class OrderCommentInline(GenericStackedInline[models.OrderComment, models.Order]):
    """
    Order comment inline
    """

    model: Type[models.OrderComment] = models.OrderComment


class AdditionalChargeInline(
    GenericStackedInline[models.AdditionalCharge, models.Order]
):
    """
    Order Additional Charge inline
    """

    model: Type[models.AdditionalCharge] = models.AdditionalCharge


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
class OrderAdmin(GenericAdmin[models.Order]):
    """
    Order Admin
    """

    list_display = (
        "pro_number",
        "status",
        "origin_location",
        "destination_location",
    )
    exclude = ()
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
                    "rate",
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
                    "auto_rate",
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
                    "voided_comm",
                )
            },
        ),
    )
    inlines = (
        OrderDocumentationInline,
        OrderCommentInline,
        AdditionalChargeInline,
    )
