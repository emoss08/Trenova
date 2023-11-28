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

from accounts.models import User
from location import models
from organization.models import BusinessUnit, Organization


@transaction.atomic
def create_or_update_location_comments(
    *,
    location: models.Location,
    location_comments_data: list[dict[str, typing.Any]],
    organization: Organization,
    business_unit: BusinessUnit,
    user: User,
) -> list[models.LocationComment]:
    created_comments = []
    if location_comments_data:
        existing_comment_id = set(
            location.location_comments.values_list("id", flat=True)
        )
        new_comment_id = set()

        for location_comment_data in location_comments_data:
            location_comment_data["business_unit"] = business_unit
            location_comment_data["organization"] = organization
            contact, created = models.LocationComment.objects.update_or_create(
                id=location_comment_data.get("id"),
                location=location,
                entered_by=user,
                defaults=location_comment_data,
            )
            created_comments.append(contact)
            if not created:
                new_comment_id.add(contact.id)

        # Delete contacts that are not in the new list
        to_delete_ids = existing_comment_id - new_comment_id
        models.LocationComment.objects.filter(id__in=to_delete_ids).delete()

    return created_comments


@transaction.atomic
def create_or_update_location_contacts(
    *,
    location: models.Location,
    location_contacts_data: list[dict[str, typing.Any]],
    organization: Organization,
    business_unit: BusinessUnit,
) -> list[models.LocationContact]:
    created_contracts = []
    if location_contacts_data:
        existing_contact_ids = set(
            location.location_contacts.values_list("id", flat=True)
        )
        new_contact_ids = set()

        for location_contact_data in location_contacts_data:
            location_contact_data["business_unit"] = business_unit
            location_contact_data["organization"] = organization
            contact, created = models.LocationContact.objects.update_or_create(
                id=location_contact_data.get("id"),
                location=location,
                defaults=location_contact_data,
            )
            created_contracts.append(contact)
            if not created:
                new_contact_ids.add(contact.id)

        # Delete contacts that are not in the new list
        to_delete_ids = existing_contact_ids - new_contact_ids
        models.LocationContact.objects.filter(id__in=to_delete_ids).delete()

    return created_contracts
