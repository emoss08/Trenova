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

from customer import models
from organization.models import BusinessUnit, Organization


def create_or_update_email_profile(
    *,
    customer: models.Customer,
    email_profile_data: dict[str, typing.Any],
    organization: Organization,
    business_unit: BusinessUnit,
) -> models.CustomerEmailProfile:
    """Create or update a customer's email profile.

    Args:
        customer (models.Customer): The customer who owns the email profile to be created or updated.
        email_profile_data (dict[str, typing.Any]): A dictionary of email profile data to be stored.
        organization (Organization): The organization to which the business unit belongs.
        business_unit (BusinessUnit): The business unit that the customer is related to.

    Returns:
        models.CustomerEmailProfile: The created or updated email profile object.
    """
    if email_profile_data:
        email_profile_data["organization"] = organization
        email_profile_data["business_unit"] = business_unit

        email_profile, _ = models.CustomerEmailProfile.objects.update_or_create(
            customer=customer, defaults=email_profile_data
        )
        return email_profile


@transaction.atomic
def create_or_update_rule_profile(
    *,
    customer: models.Customer,
    rule_profile_data: dict[str, typing.Any],
    organization: Organization,
    business_unit: BusinessUnit,
) -> models.CustomerRuleProfile:
    """Create or update a customer's rule profile.

    Args:
        customer (models.Customer): The customer who owns the rule profile to be created or updated.
        rule_profile_data (dict[str, typing.Any]): A dictionary of rule profile data to be stored.
        organization (Organization): The organization to which the business unit belongs.
        business_unit (BusinessUnit): The business unit that the customer is related to.

    Returns:
        models.CustomerRuleProfile: The created or updated rule profile object.
    """
    if rule_profile_data:
        document_classifications = rule_profile_data.pop("document_class", [])
        rule_profile_data["business_unit"] = business_unit
        rule_profile_data["organization"] = organization

        rule_profile, created = models.CustomerRuleProfile.objects.update_or_create(
            customer=customer, defaults=rule_profile_data
        )
        if document_classifications:
            rule_profile.document_class.set(document_classifications)
        return rule_profile


@transaction.atomic
def create_or_update_delivery_slots(
    *,
    customer: models.Customer,
    delivery_slots_data: list[dict[str, typing.Any]],
    organization: Organization,
    business_unit: BusinessUnit,
) -> list[models.DeliverySlot]:
    """Create or update a customer's delivery slots.

    Args:
        customer (models.Customer): The customer who owns the delivery slots to be created or updated.
        delivery_slots_data (list[dict[str, typing.Any]]): A list of dictionaries containing delivery slot data to be stored.
        organization (Organization): The organization to which the business unit belongs.
        business_unit (BusinessUnit): The business unit that the customer is related to.

    Returns:
        list[models.DeliverySlot]: A list of created or updated delivery slot objects.
    """
    created_slots = []
    if delivery_slots_data:
        existing_slot_ids = set(customer.delivery_slots.values_list("id", flat=True))
        new_slot_ids = set()

        for delivery_slot_data in delivery_slots_data:
            delivery_slot_data["business_unit"] = business_unit
            delivery_slot_data["organization"] = organization
            slot, created = models.DeliverySlot.objects.update_or_create(
                id=delivery_slot_data.get("id"),
                customer=customer,
                defaults=delivery_slot_data,
            )
            created_slots.append(slot)
            if not created:
                new_slot_ids.add(slot.id)

        # Delete slots that are not in the new list
        to_delete_ids = existing_slot_ids - new_slot_ids
        models.DeliverySlot.objects.filter(id__in=to_delete_ids).delete()

    return created_slots


@transaction.atomic
def create_or_update_customer_contacts(
    *,
    customer: models.Customer,
    customer_contacts_data: list[dict[str, typing.Any]],
    organization: Organization,
    business_unit: BusinessUnit,
) -> list[models.CustomerContact]:
    """Create or update a customer's contacts.

    Args:
        customer (models.Customer): The customer who owns the contacts to be created or updated.
        customer_contacts_data (list[dict[str, typing.Any]]): A list of dictionaries containing customer contact data to be stored.
        organization (Organization): The organization to which the business unit belongs.
        business_unit (BusinessUnit): The business unit that the customer is related to.

    Returns:
        list[models.CustomerContact]: A list of created or updated customer contact objects.
    """
    created_contacts = []
    if customer_contacts_data:
        existing_contact_ids = set(customer.contacts.values_list("id", flat=True))
        new_contact_ids = set()

        for customer_contact_data in customer_contacts_data:
            customer_contact_data["business_unit"] = business_unit
            customer_contact_data["organization"] = organization
            contact, created = models.CustomerContact.objects.update_or_create(
                id=customer_contact_data.get("id"),
                customer=customer,
                defaults=customer_contact_data,
            )
            created_contacts.append(contact)
            if not created:
                new_contact_ids.add(contact.id)

        # Delete contacts that are not in the new list
        to_delete_ids = existing_contact_ids - new_contact_ids
        models.CustomerContact.objects.filter(id__in=to_delete_ids).delete()

    return created_contacts
