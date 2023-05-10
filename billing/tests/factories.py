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

import factory


class ChargeTypeFactory(factory.django.DjangoModelFactory):
    """
    ChargeType factory
    """

    class Meta:
        """
        Metaclass for ChargeTypeFactory
        """

        model = "billing.ChargeType"
        django_get_or_create = ("organization",)

    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    name = factory.Faker("word", locale="en_US")


class AccessorialChargeFactory(factory.django.DjangoModelFactory):
    """
    AccessorialCharge factory
    """

    class Meta:
        """
        Metaclass for AccessorialChargeFactory
        """

        model = "billing.AccessorialCharge"
        django_get_or_create = ("organization",)

    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    code = factory.Faker("word", locale="en_US")
    charge_amount = 100.0


class DocumentClassificationFactory(factory.django.DjangoModelFactory):
    """
    DocumentClassification factory
    """

    class Meta:
        """
        Metaclass for DocumentClassificationFactory
        """

        model = "billing.DocumentClassification"
        django_get_or_create = ("organization",)

    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    name = factory.Faker("word", locale="en_US")


class BillingQueueFactory(factory.django.DjangoModelFactory):
    """
    Billing Queue Factory
    """

    class Meta:
        model = "billing.BillingQueue"
        django_get_or_create = ("order", "organization", "customer")

    order = factory.SubFactory("order.tests.factories.OrderFactory")
    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    customer = factory.SubFactory("customer.factories.CustomerFactory")
