# --------------------------------------------------------------------------------------------------
#  COPYRIGHT(c) 2023 MONTA                                                                         -
#                                                                                                  -
#  This file is part of Monta.                                                                     -
#                                                                                                  -
#  Monta is free software: you can redistribute it and/or modify                                   -
#  it under the terms of the GNU General Public License as published by                            -
#  the Free Software Foundation, either version 3 of the License, or                               -
#  (at your option) any later version.                                                             -
#                                                                                                  -
#  Monta is distributed in the hope that it will be useful,                                        -
#  but WITHOUT ANY WARRANTY; without even the implied warranty of                                  -
#  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the                                   -
#  GNU General Public License for more details.                                                    -
#                                                                                                  -
#  You should have received a copy of the GNU General Public License                               -
#  along with Monta.  If not, see <https://www.gnu.org/licenses/>.                                 -
# --------------------------------------------------------------------------------------------------

from django.contrib import admin

from customer import models
from utils.admin import GenericAdmin, GenericStackedInline


@admin.register(models.CustomerEmailProfile)
class CustomerEmailProfileAdmin(GenericAdmin[models.CustomerEmailProfile]):
    """
    Customer Email Profile Admin
    """

    model: type[models.CustomerEmailProfile] = models.CustomerEmailProfile
    list_display = (
        "id",
        "name",
    )
    search_fields = ("id",)


@admin.register(models.CustomerRuleProfile)
class CustomerRuleProfileAdmin(GenericAdmin[models.CustomerRuleProfile]):
    """
    Customer Rule Profile Admin
    """

    model: type[models.CustomerRuleProfile] = models.CustomerRuleProfile
    list_display = ("name",)
    search_fields = ("name",)


@admin.register(models.CustomerBillingProfile)
class CustomerBillingProfileAdmin(GenericAdmin[models.CustomerBillingProfile]):
    """
    Customer Billing Profile Admin
    """

    model: type[models.CustomerBillingProfile] = models.CustomerBillingProfile
    list_display = ("customer",)
    search_fields = ("customer",)


class CustomerFuelTableDetailInline(
    GenericStackedInline[models.CustomerFuelTable, models.CustomerFuelTableDetail]
):
    """
    Customer Fuel Table Detail
    """

    model: type[models.CustomerFuelTableDetail] = models.CustomerFuelTableDetail
    extra = 10
    verbose_name_plural = "Customer Fuel Details"
    fk_name = "customer_fuel_table"


@admin.register(models.CustomerFuelTable)
class CustomerFuelTableAdmin(GenericAdmin[models.CustomerFuelTable]):
    """
    Customer Fuel Table Admin
    """

    model: type[models.CustomerFuelTable] = models.CustomerFuelTable
    list_display = (
        "id",
        "description",
    )
    search_fields: tuple[str, ...] = ("id",)
    inlines = (CustomerFuelTableDetailInline,)


@admin.register(models.CustomerFuelProfile)
class CustomerFuelProfileAdmin(GenericAdmin[models.CustomerFuelProfile]):
    """
    Customer Fuel Profile Admin
    """

    model: type[models.CustomerFuelProfile] = models.CustomerFuelProfile
    list_display = (
        "id",
        "customer",
    )
    search_fields: tuple[str, ...] = ("id",)


class CustomerContactInline(
    GenericStackedInline[models.Customer, models.CustomerContact]
):
    """
    Customer Contact
    """

    model: type[models.CustomerContact] = models.CustomerContact
    fk_name = "customer"


@admin.register(models.Customer)
class CustomerAdmin(GenericAdmin[models.Customer]):
    """
    Customer Admin
    """

    model: type[models.Customer] = models.Customer
    list_display = (
        "code",
        "name",
    )
    search_fields = ("name",)
    inlines = (CustomerContactInline,)
