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
from django.db.models import QuerySet
from django.utils import timezone
from order.models import Order, AdditionalCharge
from dispatch import models
from order.selectors import sum_order_additional_charges


def get_rate(*, order: Order) -> models.Rate | None:
    """Get the rate for the order.

    Args:
        order(Order): The order to get the rate for.

    Returns:
        models.Rate | None: The rate for the order or None if no rate is found.
    """
    today = timezone.now().date()
    rates = models.Rate.objects.filter(
        customer=order.customer,
        commodity=order.commodity,
        order_type=order.order_type,
        equipment_type=order.equipment_type,
        origin_location=order.origin_location,
        destination_location=order.destination_location,
        effective_date__lte=today,
        expiration_date__gte=today,
    )
    return rates.first() if rates.exists() else None


def get_rate_billing_table(
    *, rate: models.Rate
) -> QuerySet[models.RateBillingTable] | None:
    return models.RateBillingTable.objects.filter(rate=rate).all() if rate else None


def transfer_rate_details(order: Order) -> None:
    """Transfer rate details to the order.

    Args:
        order (Order): The order to transfer rate details to.

    Returns:
        None: This function does not return anything.
    """

    if rate := get_rate(order=order):
        order.freight_charge_amount = rate.rate_amount
        order.mileage = rate.distance_override

    for billing_item in get_rate_billing_table(rate=rate):
        # Check if the charge already exists on the order
        additional_charge_exists = AdditionalCharge.objects.filter(
            organization=order.organization,
            order=order,
            accessorial_charge=billing_item.accessorial_charge,
        ).exists()

        if not additional_charge_exists:
            AdditionalCharge.objects.create(
                organization=order.organization,
                order=order,
                accessorial_charge=billing_item.accessorial_charge,
                charge_amount=billing_item.charge_amount,
                unit=billing_item.unit,
                description=billing_item.description,
                entered_by=order.entered_by,
            )

    order.other_charge_amount = sum_order_additional_charges(order=order)
