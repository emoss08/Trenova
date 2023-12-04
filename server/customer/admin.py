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

from django.contrib import admin

from customer import models
from utils.admin import GenericAdmin, GenericStackedInline


class CustomerEmailProfileAdmin(
    GenericStackedInline[models.CustomerEmailProfile, models.Customer]
):
    """
    Customer Email Profile Admin
    """

    model = models.CustomerEmailProfile
    list_display = (
        "id",
        "customer",
    )
    search_fields = ("id",)


class CustomerRuleProfileAdmin(
    GenericStackedInline[models.CustomerRuleProfile, models.Customer]
):
    """
    Customer Rule Profile Admin
    """

    model = models.CustomerRuleProfile
    list_display = ("name",)
    search_fields = ("name",)


class CustomerFuelTableDetailInline(
    GenericStackedInline[models.CustomerFuelTable, models.CustomerFuelTableDetail]
):
    """
    Customer Fuel Table Detail
    """

    model = models.CustomerFuelTableDetail
    extra = 10
    verbose_name_plural = "Customer Fuel Details"
    fk_name = "customer_fuel_table"


@admin.register(models.CustomerFuelTable)
class CustomerFuelTableAdmin(GenericAdmin[models.CustomerFuelTable]):
    """
    Customer Fuel Table Admin
    """

    model = models.CustomerFuelTable
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

    model = models.CustomerFuelProfile
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


class DeliverySlotInline(GenericStackedInline[models.Customer, models.DeliverySlot]):
    """
    Delivery Slot
    """

    model = models.DeliverySlot
    fk_name = "customer"


@admin.register(models.Customer)
class CustomerAdmin(GenericAdmin[models.Customer]):
    """
    Customer Admin
    """

    model = models.Customer
    list_display = (
        "code",
        "name",
    )
    search_fields = ("name",)
    inlines = (
        CustomerContactInline,
        CustomerRuleProfileAdmin,
        CustomerEmailProfileAdmin,
        DeliverySlotInline,
    )
