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

from order.models import commodity, hazardous_material, movement, order, order_control, qualifier_code, reason_code, \
    service_incident, stop
from utils.admin import GenericAdmin, GenericStackedInline, GenericTabularInline


class OrderDocumentationInline(GenericTabularInline):
    """
    Order documentation inline
    """

    model: type[order.OrderDocumentation] = order.OrderDocumentation


class OrderComment(GenericStackedInline):
    """
    Order comment inline
    """

    model: type[order.OrderComment] = order.OrderComment


@admin.register(order.OrderType)
class OrderTypeAdmin(GenericAdmin[order.OrderType]):
    """
    Order Type Admin
    """

    list_display = (
        "name",
        "description",
    )
    search_fields = ("name", "description")


@admin.register(hazardous_material.HazardousMaterial)
class HazardousMaterialAdmin(GenericAdmin[hazardous_material.HazardousMaterial]):
    """
    Hazardous Material Admin
    """

    list_display = (
        "name",
        "description",
    )
    search_fields = ("name", "description")


@admin.register(commodity.Commodity)
class CommodityAdmin(GenericAdmin[commodity.Commodity]):
    """
    Commodity Admin
    """

    list_display = (
        "name",
        "description",
    )
    search_fields = ("name", "description")
    fieldsets = (
        (None, {"fields": ("name", "description")}),
        (
            "Hazmat Information",
            {
                "classes": ("collapse",),
                "fields": (
                    "min_temp",
                    "max_temp",
                    "set_point_temp",
                    "unit_of_measure",
                    "hazmat",
                    "is_hazmat",
                ),
            },
        ),
    )


@admin.register(qualifier_code.QualifierCode)
class QualifierCodeAdmin(GenericAdmin[qualifier_code.QualifierCode]):
    """
    Qualifier Code Admin
    """

    list_display = (
        "code",
        "description",
    )
    search_fields = ("code", "description")


@admin.register(reason_code.ReasonCode)
class ReasonCodeAdmin(GenericAdmin[reason_code.ReasonCode]):
    """
    Reason Code Admin
    """

    list_display = (
        "code",
        "description",
    )
    search_fields = ("code", "description")


@admin.register(order_control.OrderControl)
class OrderControlAdmin(GenericAdmin[order_control.OrderControl]):
    """
    Order Control Admin
    """

    list_display = (
        "organization",
        "auto_rate_orders",
    )
    search_fields = ("organization", "auto_rate_orders")


@admin.register(order.Order)
class OrderAdmin(GenericAdmin[order.Order]):
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
        (None, {"fields": ("status", "revenue_code", "entered_by")}),
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
                    "hazmat_id",
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
        OrderDocumentationInline,
        OrderComment,
    )


@admin.register(movement.Movement)
class MovementAdmin(GenericAdmin[movement.Movement]):
    """
    Movement Admin
    """

    list_display = (
        "status",
        "ref_num",
        "order",
        "equipment",
        "primary_worker",
    )
    search_fields = ("ref_num",)


@admin.register(stop.Stop)
class StopAdmin(GenericAdmin[stop.Stop]):
    """
    Stop Admin
    """

    list_display = (
        "status",
        "movement",
        "stop_type",
        "sequence",
        "location",
        "address_line",
    )
    search_fields = ("id",)


@admin.register(service_incident.ServiceIncident)
class ServiceIncidentAdmin(GenericAdmin[service_incident.ServiceIncident]):
    """
    Service Incident Admin
    """

    list_display = (
        "movement",
        "stop",
        "delay_code",
        "delay_reason",
        "delay_time",
    )
    search_fields = ("id",)
