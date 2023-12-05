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

import typing

from django.db import transaction

from dispatch import models
from organization.models import BusinessUnit, Organization


@transaction.atomic
def create_or_update_rate_billing_table(
    *,
    rate: models.Rate,
    rate_billing_tables_data: list[dict[str, typing.Any]],
    organization: Organization,
    business_unit: BusinessUnit,
) -> list[models.RateBillingTable]:
    """This function creates a new rate billing table or updates an existing one according to the provided parameters.

    This function is decorated with `transaction.atomic` to ensure that database operations within
    the function all succeed or all fail together, keeping the database in a consistent state.

    Args:
        rate (models.Rate): The rate object to create or update in the billing table.
        rate_billing_tables_data (dict[str, typing.Any]): A dictionary containing various details related to
            the billing table. Must include at least "organization" and "business_unit" as keys.
        organization (Organization): The organization object. This object is added to rate_profile_data
            before updating the database.
        business_unit (BusinessUnit): The business unit object. This object is added to rate_profile_data
            before updating the database.

    Returns:
        list[models.RateBillingTable]: A list of created or updated Rate Billing Table objects.
    """
    created_table_data = []
    existing_table_ids = set(rate.rate_billing_tables.values_list("id", flat=True))
    new_table_ids = set()

    for rate_table_data in rate_billing_tables_data:
        rate_table_data["business_unit"] = business_unit
        rate_table_data["organization"] = organization
        rate_table, created = models.RateBillingTable.objects.update_or_create(
            id=rate_table_data.get("id"),
            rate=rate,
            defaults=rate_table_data,
        )
        created_table_data.append(rate_table)
        if not created:
            new_table_ids.add(rate_table.id)

    # Delete any rate billing tables that are not in the new list
    to_delete_id = existing_table_ids - new_table_ids

    models.RateBillingTable.objects.filter(id__in=to_delete_id).delete()

    return created_table_data
