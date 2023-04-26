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

import datetime

import factory
from django.utils import timezone


class DispatchControlFactory(factory.django.DjangoModelFactory):
    """
    Dispatch control factory
    """

    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    regulatory_check = True

    class Meta:
        """
        Metaclass for DispatchControlFactory
        """

        model = "dispatch.DispatchControl"
        django_get_or_create = ("organization",)


class DelayCodeFactory(factory.django.DjangoModelFactory):
    """
    Delay code factory
    """

    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    code = factory.Faker("pystr", max_chars=4)
    description = factory.Faker("text", locale="en_US", max_nb_chars=10)

    class Meta:
        """
        Metaclass for DelayCodeFactory
        """

        model = "dispatch.DelayCode"
        django_get_or_create = ("organization",)


class FleetCodeFactory(factory.django.DjangoModelFactory):
    """
    Fleet code factory
    """

    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    code = factory.Faker("pystr", max_chars=4)
    description = factory.Faker("text", locale="en_US", max_nb_chars=10)

    class Meta:
        """
        Metaclass for FleetCodeFactory
        """

        model = "dispatch.FleetCode"
        django_get_or_create = ("organization",)


class CommentTypeFactory(factory.django.DjangoModelFactory):
    """
    Comment type factory
    """

    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    name = factory.Faker("word", locale="en_US")
    description = factory.Faker("text", locale="en_US", max_nb_chars=10)

    class Meta:
        """
        Metaclass for CommentTypeFactory
        """

        model = "dispatch.CommentType"
        django_get_or_create = ("organization",)


class RateFactory(factory.django.DjangoModelFactory):
    """
    Rate Factory
    """

    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    customer = factory.SubFactory("customer.factories.CustomerFactory")
    effective_date = timezone.now().date()
    expiration_date = timezone.now().date() + datetime.timedelta(days=365)
    commodity = factory.SubFactory("commodities.factories.CommodityFactory")
    order_type = factory.SubFactory("order.tests.factories.OrderTypeFactory")
    equipment_type = factory.SubFactory(
        "equipment.tests.factories.EquipmentTypeFactory"
    )
    rate_amount = 1_000_000.00

    class Meta:
        """
        Metaclass for RateFactory
        """

        model = "dispatch.Rate"
        django_get_or_create = (
            "organization",
            "order_type",
            "equipment_type",
        )


class RateBillingTableFactory(factory.django.DjangoModelFactory):
    """
    Rate Billing Table Factory
    """

    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    rate = factory.SubFactory(RateFactory)
    accessorial_charge = factory.SubFactory("billing.tests.factories.AccessorialChargeFactory")
    description = factory.Faker("text", locale="en_US", max_nb_chars=100)
    unit = 1
    charge_amount = 100.00

    class Meta:
        """
        Metaclass for RateBillingTableFactory
        """

        model = "dispatch.RateBillingTable"
        django_get_or_create = (
            "organization",
            "rate",
            "accessorial_charge",
        )
