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

from core.mixins import MontaAdminMixin, MontaTabularInlineMixin
from order import models


class OrderDocumentationInline(MontaTabularInlineMixin):
    """
    Order documentation inline
    """

    model: type[models.OrderDocumentation] = models.OrderDocumentation
    extra = 0


@admin.register(models.OrderType)
class OrderTypeAdmin(MontaAdminMixin[models.OrderType]):
    """
    Order Type Admin
    """

    list_display = (
        "name",
        "description",
    )
    search_fields = ("name", "description")


@admin.register(models.HazardousMaterial)
class HazardousMaterialAdmin(MontaAdminMixin[models.HazardousMaterial]):
    """
    Hazardous Material Admin
    """

    list_display = (
        "name",
        "description",
    )
    search_fields = ("name", "description")


@admin.register(models.Commodity)
class CommodityAdmin(MontaAdminMixin[models.Commodity]):
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


@admin.register(models.QualifierCode)
class QualifierCodeAdmin(MontaAdminMixin[models.QualifierCode]):
    """
    Qualifier Code Admin
    """

    list_display = (
        "code",
        "description",
    )
    search_fields = ("code", "description")


@admin.register(models.ReasonCode)
class ReasonCodeAdmin(MontaAdminMixin[models.ReasonCode]):
    """
    Reason Code Admin
    """

    list_display = (
        "code",
        "description",
    )
    search_fields = ("code", "description")


@admin.register(models.OrderControl)
class OrderControlAdmin(MontaAdminMixin[models.OrderControl]):
    """
    Order Control Admin
    """

    list_display = (
        "organization",
        "auto_rate_orders",
    )


@admin.register(models.Order)
class OrderAdmin(MontaAdminMixin[models.Order]):
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
    inlines = (OrderDocumentationInline,)


@admin.register(models.Movement)
class MovementAdmin(MontaAdminMixin[models.Movement]):
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


@admin.register(models.Stop)
class StopAdmin(MontaAdminMixin[models.Stop]):
    """
    Stop Admin
    """

    list_display = (
        "status",
        "movement",
        "sequence",
        "location",
    )
    search_fields = ("id",)


@admin.register(models.ServiceIncident)
class ServiceIncidentAdmin(MontaAdminMixin[models.ServiceIncident]):
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
