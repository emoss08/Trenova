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

from organization.models import Organization, BusinessUnit
from customer import models
import typing


def create_or_update_email_profile(
    *,
    customer: models.Customer,
    email_profile_data: dict[str, typing.Any],
    organization: Organization,
    business_unit: BusinessUnit,
) -> None:
    """Creates or updates the email profile of a customer.

    This function accepts a customer instance, a dictionary containing the email profile data,
    an organization instance, and a business unit instance as arguments. It then associates the
    business unit and the organization with the email profile data and uses it to create or update
    the customer's email profile.

    If the email profile already exists, it is updated with the new data. If it does not exist,
    a new instance is created.

    Args:
        customer (models.Customer): The customer for whom the email profile is to be created or updated.
        email_profile_data (dict[str, typing.Any]): A dictionary containing the email profile data.
        organization (Organization): The organization that the customer belongs to.
        business_unit (BusinessUnit): The business unit that the customer belongs to.

    Returns:
        None: This function does not return anything.
    """
    if email_profile_data:
        email_profile_data["business_unit"] = business_unit
        email_profile_data["organization"] = organization
        models.CustomerEmailProfile.objects.update_or_create(
            customer=customer, defaults=email_profile_data
        )


def create_or_update_rule_profile(
    *,
    customer: models.Customer,
    rule_profile_data: dict[str, typing.Any],
    organization: Organization,
    business_unit: BusinessUnit,
) -> None:
    """Creates or updates the rule profile of a customer.

    It takes a customer instance, a dictionary containing rule profile data, an organization instance,
    and a business unit instance as arguments. It then associates the business unit, the organization,
    and the rule profile data with the customer to create or update the rule profile.

    If the rule profile already exists, it is updated. If it does not exist, a new one is created.

    Args:
        customer (models.Customer): The customer for whom the rule profile is to be created or updated.
        rule_profile_data (dict[str, typing.Any]): A dictionary containing the rule profile data.
        organization (Organization): The organization that the customer belongs to.
        business_unit (BusinessUnit): The business unit that the customer belongs to.

    Returns:
        None: This function does not return anything.
    """
    if rule_profile_data:
        document_classifications = rule_profile_data.pop("document_class", [])
        rule_profile_data["business_unit"] = business_unit
        rule_profile_data["organization"] = organization
        (
            rule_profile,
            created,
        ) = models.CustomerRuleProfile.objects.update_or_create(
            customer=customer, defaults=rule_profile_data
        )
        if document_classifications:
            rule_profile.document_class.set(document_classifications)


def create_or_update_delivery_slots(
    customer: models.Customer,
    delivery_slots_data: list[dict[str, typing.Any]],
    organization: Organization,
    business_unit: BusinessUnit,
) -> None:
    """Creates or updates the delivery slots for a customer.

    This function accepts a customer instance, a list of dictionaries each representing a delivery
    slot, an organization instance, and a business unit instance as arguments. It then associates
    the business unit and the organization with each delivery slot and uses the data to create them.
    Existing delivery slots for the customer are first deleted before the new ones are created.

    Args:
        customer (models.Customer): The customer for whom the delivery slots are to be created or updated.
        delivery_slots_data (list[dict[str, typing.Any]]): A list of dictionaries, each dictionary
        representing a delivery slot.
        organization (Organization): The organization that the customer belongs to.
        business_unit (BusinessUnit): The business unit that the customer belongs to.

    Returns:
        None: This function does not return anything.
    """
    if delivery_slots_data:
        models.DeliverySlot.objects.filter(customer=customer).delete()
        for delivery_slot_data in delivery_slots_data:
            delivery_slot_data["business_unit"] = business_unit
            delivery_slot_data["organization"] = organization
        models.DeliverySlot.objects.bulk_create(
            [
                models.DeliverySlot(customer=customer, **delivery_slot_data)
                for delivery_slot_data in delivery_slots_data
            ]
        )


def create_or_update_customer_contacts(
    customer: models.Customer,
    customer_contacts_data: list[dict[str, typing.Any]],
    organization: Organization,
    business_unit: BusinessUnit,
) -> None:
    """Creates or updates the contacts for a customer.

    This function accepts a customer instance, a list of dictionaries each representing a contact,
    an organization instance, and a business unit instance. It then associates the business unit and
    the organization with each contact and uses the data to create these contacts.
    Existing contacts for the customer are first deleted before the new ones are created.

    Args:
        customer (models.Customer): The customer for whom the contacts are to be created or updated.
        customer_contacts_data (list[dict[str, typing.Any]]): A list of dictionaries, each dictionary representing
        a contact.
        organization (Organization): The organization that the customer belongs to.
        business_unit (BusinessUnit): The business unit that the customer belongs to.

    Returns:
        None: This function does not return Anything.
    """
    if customer_contacts_data:
        models.CustomerContact.objects.filter(customer=customer).delete()
        for customer_contact_data in customer_contacts_data:
            customer_contact_data["business_unit"] = business_unit
            customer_contact_data["organization"] = organization
        models.CustomerContact.objects.bulk_create(
            [
                models.CustomerContact(customer=customer, **customer_contact_data)
                for customer_contact_data in customer_contacts_data
            ]
        )
