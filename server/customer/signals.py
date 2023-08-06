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

from customer import models, services
from utils.models import PrimaryStatusChoices


def generate_customer_code(instance: models.Customer, **kwargs: Any) -> None:
    """
    Generate a unique customer code for a new or existing customer instance.

    This function is designed to be used as a Django signal receiver. It will
    generate a customer code using the CustomerGenerationService and assign it
    to the instance if it does not already have one.

    Args:
        instance (models.Customer): The instance of the Customer model for which
            the code is being generated.
        **kwargs (Any): Additional keyword arguments passed by the signal.

    Returns:
        None
    """
    if not instance.code:
        instance.code = services.generate_customer_code(instance=instance)


def create_customer_billing_profile(
    instance: models.Customer, created: bool, **kwargs: Any
) -> None:
    """
    Create a billing profile for a new customer if it does not already exist.

    This function is designed to be used as a Django signal receiver. It will
    create a billing profile for a new customer instance if it does not already
    have one, and if the default rule profile exists. The billing profile
    will be set as active and will be associated with the default rule profile.

    Args:
        instance (models.Customer): The instance of the Customer model for which
            the billing profile is being created.
        created (bool): Indicates whether the instance is being created or updated.
        **kwargs (Any): Additional keyword arguments passed by the signal.

    Returns:
        None
    """
    (
        default_rule_profile,
        rule_profile_exists,
    ) = models.CustomerRuleProfile.objects.get_or_create(
        business_unit=instance.organization.business_unit,
        organization=instance.organization,
        name="Default",
    )

    if (
        not models.CustomerBillingProfile.objects.filter(customer=instance).exists()
        and rule_profile_exists
        and created
    ):
        models.CustomerBillingProfile.objects.create(
            business_unit=instance.organization.business_unit,
            organization=instance.organization,
            customer=instance,
            status=PrimaryStatusChoices.ACTIVE,
            rule_profile=default_rule_profile,
        )
