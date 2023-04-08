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

from typing import Any

from dispatch.models import Rate, RateBillingTable


def set_rate_number(instance: Rate, **kwargs: Any) -> None:
    """
    Set the rate_number field of a Rate instance before it is created.

    This method sets the rate_number field of the Rate instance to the result of the `generate_rate_number` method.

    Returns:
        None
    """
    if not instance.rate_number:
        instance.rate_number = Rate.generate_rate_number()


def set_charge_amount_on_billing_table(
    instance: RateBillingTable, **kwargs: Any
) -> None:
    """
    Set the charge amount for the rate billing table instance.

    Returns:
        None: None
    """
    if not instance.charge_amount:
        instance.charge_amount = instance.charge_code.charge_amount

    instance.sub_total = instance.charge_amount * instance.units
