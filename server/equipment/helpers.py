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

from equipment import models

if typing.TYPE_CHECKING:
    from organization.models import BusinessUnit, Organization


@transaction.atomic
def create_or_update_equip_type_details(
    *,
    equipment_type: models.EquipmentType,
    business_unit: "BusinessUnit",
    organization: "Organization",
    detail_data: dict[str, typing.Any],
) -> models.EquipmentTypeDetail | None:
    """Create or update equipment type details.

    Args:
        equipment_type (models.EquipmentType): The equipment type who owns the details to be created or updated.
        detail_data (list[dict[str, typing.Any]]): A list of dictionaries containing equipment type details
        data to be stored.
        organization (Organization): The organization to which the business unit belongs.
        business_unit (BusinessUnit): The business unit that the worker is related to.

    Returns:
        models.EquipmentTypeDetail: A list of created or updated equipment type details
        objects.
    """

    if detail_data:
        detail_data["organization"] = organization
        detail_data["business_unit"] = business_unit

        details, _ = models.EquipmentTypeDetail.objects.update_or_create(
            equipment_type=equipment_type, defaults=detail_data
        )
        return details

    return None
