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

from core.generics.admin import GenericAdmin, GenericStackedInline

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
class CustomerEmailProfileAdmin(GenericAdmin[CustomerEmailProfile]):
    """
    Customer Email Profile Admin
    """

    model: Type[CustomerEmailProfile] = CustomerEmailProfile
    list_display = (
        "id",
        "name",
    )
    search_fields = ("id",)


@admin.register(CustomerRuleProfile)
class CustomerRuleProfileAdmin(GenericAdmin[CustomerRuleProfile]):
    """
    Customer Rule Profile Admin
    """

    model: Type[CustomerRuleProfile] = CustomerRuleProfile
    list_display = ("name",)
    search_fields = ("name",)


class CustomerBillingProfileInline(GenericStackedInline[CustomerBillingProfile]):
    """
    Customer Billing Profile
    """

    model: Type[CustomerBillingProfile] = CustomerBillingProfile
    extra = 0
    can_delete = False
    verbose_name_plural = "Billing Profiles"
    fk_name = "customer"
    exclude = ("organization",)


class CustomerFuelTableDetailInline(GenericStackedInline[CustomerFuelTableDetail]):
    """
    Customer Fuel Table Detail
    """

    model: Type[CustomerFuelTableDetail] = CustomerFuelTableDetail
    extra = 10
    verbose_name_plural = "Customer Fuel Details"
    fk_name = "customer_fuel_table"


@admin.register(CustomerFuelTable)
class CustomerFuelTableAdmin(GenericAdmin[CustomerFuelTable]):
    """
    Customer Fuel Table Admin
    """

    model: Type[CustomerFuelTable] = CustomerFuelTable
    list_display = (
        "id",
        "description",
    )
    search_fields: tuple[str, ...] = ("id",)
    inlines = (CustomerFuelTableDetailInline,)


@admin.register(CustomerFuelProfile)
class CustomerFuelProfileAdmin(GenericAdmin[CustomerFuelProfile]):
    """
    Customer Fuel Profile Admin
    """

    model: Type[CustomerFuelProfile] = CustomerFuelProfile
    list_display = (
        "id",
        "customer",
    )
    search_fields: tuple[str, ...] = ("id",)


class CustomerContactInline(GenericStackedInline[CustomerContact]):
    """
    Customer Contact
    """

    model: Type[CustomerContact] = CustomerContact
    extra = 0
    verbose_name_plural = "Customer Contacts"
    fk_name = "customer"
    exclude = ("organization",)


@admin.register(Customer)
class CustomerAdmin(GenericAdmin[Customer]):
    """
    Customer Admin
    """

    model: Type[Customer] = Customer
    list_display = (
        "code",
        "name",
    )
    search_fields = ("name",)
    inlines = (CustomerBillingProfileInline, CustomerContactInline)
