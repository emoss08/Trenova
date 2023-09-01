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

from organization.models import BusinessUnit, Organization
from worker import models


@transaction.atomic
def create_or_update_worker_profile(
    *,
    worker: models.Worker,
    profile_data: dict[str, typing.Any],
    organization: Organization,
    business_unit: BusinessUnit,
) -> models.WorkerProfile | None:
    """Create or update a worker's profile.

    Args:
        worker (models.Worker): The worker who owns the profile to be created or updated.
        profile_data (dict[str, typing.Any]): A dictionary of profile data to be stored.
        organization (Organization): The organization to which the business unit belongs.
        business_unit (BusinessUnit): The business unit that the worker is related to.

    Returns:
        models.WorkerProfile: The created or updated WorkerProfile object.
    """

    if profile_data:
        profile_data["organization"] = organization
        profile_data["business_unit"] = business_unit

        profile, _ = models.WorkerProfile.objects.update_or_create(
            worker=worker, defaults=profile_data
        )
        return profile

    return None


@transaction.atomic
def create_or_update_worker_contacts(
    *,
    worker: models.Worker,
    worker_contacts_data: list[dict[str, typing.Any]],
    organization: Organization,
    business_unit: BusinessUnit,
) -> list[models.WorkerContact]:
    """Create or update a worker's contacts.

    Args:
        worker (models.Worker): The worker who owns the contacts to be created or updated.
        worker_contacts_data (list[dict[str, typing.Any]]): A list of dictionaries containing worker contact data to be stored.
        organization (Organization): The organization to which the business unit belongs.
        business_unit (BusinessUnit): The business unit that the worker is related to.

    Returns:
        list[models.WorkerContact]: A list of created or updated WorkerContact objects.
    """

    created_contacts = []
    if worker_contacts_data:
        existing_contact_ids = set(worker.contacts.values_list("id", flat=True))
        new_contact_ids = set()

        for worker_contact_data in worker_contacts_data:
            worker_contact_data["business_unit"] = business_unit
            worker_contact_data["organization"] = organization
            contact, created = models.WorkerContact.objects.update_or_create(
                id=worker_contact_data.get("id"),
                worker=worker,
                defaults=worker_contact_data,
            )
            created_contacts.append(contact)
            if not created:
                new_contact_ids.add(contact.id)

        # Delete contacts that are not in the new list
        to_delete_ids = existing_contact_ids - new_contact_ids
        models.WorkerContact.objects.filter(id__in=to_delete_ids).delete()

    return created_contacts


@transaction.atomic
def create_or_update_worker_comments(
    *,
    worker: models.Worker,
    worker_comment_data: list[dict[str, typing.Any]],
    organization: Organization,
    business_unit: BusinessUnit,
) -> list[models.WorkerComment]:
    """Create or update a worker's comments.

    Args:
        worker (models.Worker): The worker who owns the comments to be created or updated.
        worker_comment_data (list[dict[str, typing.Any]]): A list of dictionaries containing worker comment data to be stored.
        organization (Organization): The organization to which the business unit belongs.
        business_unit (BusinessUnit): The business unit that the worker is related to.

    Returns:
        list[models.WorkerComment]: A list of created or updated WorkerComment objects.
    """

    created_comments = []
    if worker_comment_data:
        existing_comment_ids = set(worker.comments.values_list("id", flat=True))
        new_comment_ids = set()

        for comment_data in worker_comment_data:
            comment_data["organization"] = organization
            comment_data["business_unit"] = business_unit
            comment, created = models.WorkerComment.objects.update_or_create(
                id=comment_data.get("id"), worker=worker, defaults=comment_data
            )
            created_comments.append(comment)
            if not created:
                new_comment_ids.add(comment.id)

        # Delete comments that are not in the new list
        to_delete_ids = existing_comment_ids - new_comment_ids
        models.WorkerComment.objects.filter(id__in=to_delete_ids).delete()

    return created_comments
