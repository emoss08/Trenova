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

from core.mixins import MontaAdminMixin, MontaStackedInlineMixin
from .models import (
    Customer,
    CustomerBillingProfile,
    CustomerContact,
    CustomerEmailProfile,
    CustomerFuelProfile,
    CustomerFuelTable,
    CustomerFuelTableDetail,
    CustomerRuleProfile,
)


@admin.register(CustomerEmailProfile)
class CustomerEmailProfileAdmin(MontaAdminMixin[CustomerEmailProfile]):
    """
    Customer Email Profile Admin
    """

    model: type[CustomerEmailProfile] = CustomerEmailProfile
    list_display = (
        "id",
        "name",
    )
    search_fields = ("id",)


@admin.register(CustomerRuleProfile)
class CustomerRuleProfileAdmin(MontaAdminMixin[CustomerRuleProfile]):
    """
    Customer Rule Profile Admin
    """

    model: type[CustomerRuleProfile] = CustomerRuleProfile
    list_display = ("name",)
    search_fields = ("name",)


class CustomerBillingProfileInline(
    MontaStackedInlineMixin[Customer, CustomerBillingProfile]
):
    """
    Customer Billing Profile
    """

    model: type[CustomerBillingProfile] = CustomerBillingProfile
    extra = 0
    can_delete = False
    verbose_name_plural = "Billing Profiles"
    fk_name = "customer"
    exclude = ("organization",)


class CustomerFuelTableDetailInline(
    MontaStackedInlineMixin[CustomerFuelTable, CustomerFuelTableDetail]
):
    """
    Customer Fuel Table Detail
    """

    model: type[CustomerFuelTableDetail] = CustomerFuelTableDetail
    extra = 10
    verbose_name_plural = "Customer Fuel Details"
    fk_name = "customer_fuel_table"


@admin.register(CustomerFuelTable)
class CustomerFuelTableAdmin(MontaAdminMixin[CustomerFuelTable]):
    """
    Customer Fuel Table Admin
    """

    model: type[CustomerFuelTable] = CustomerFuelTable
    list_display = (
        "id",
        "description",
    )
    search_fields: tuple[str, ...] = ("id",)
    inlines = (CustomerFuelTableDetailInline,)


@admin.register(CustomerFuelProfile)
class CustomerFuelProfileAdmin(MontaAdminMixin[CustomerFuelProfile]):
    """
    Customer Fuel Profile Admin
    """

    model: type[CustomerFuelProfile] = CustomerFuelProfile
    list_display = (
        "id",
        "customer",
    )
    search_fields: tuple[str, ...] = ("id",)


class CustomerContactInline(MontaStackedInlineMixin[Customer, CustomerContact]):
    """
    Customer Contact
    """

    model: type[CustomerContact] = CustomerContact
    extra = 0
    verbose_name_plural = "Customer Contacts"
    fk_name = "customer"
    exclude = ("organization",)


@admin.register(Customer)
class CustomerAdmin(MontaAdminMixin[Customer]):
    """
    Customer Admin
    """

    model: type[Customer] = Customer
    list_display = (
        "code",
        "name",
    )
    search_fields = ("name",)
    inlines = (CustomerBillingProfileInline, CustomerContactInline)
