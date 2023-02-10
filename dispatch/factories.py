"""
COPYRIGHT 2022 MONTA

This file is part of Monta.

Monta is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

Monta is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with Monta.  If not, see <https://www.gnu.org/licenses/>.
"""

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
    expiration_date = timezone.now().date() + timezone.timedelta(days=365)
    commodity = factory.SubFactory("commodities.factories.CommodityFactory")
    order_type = factory.SubFactory("order.tests.factories.OrderTypeFactory")
    equipment_type = factory.SubFactory(
        "equipment.tests.factories.EquipmentTypeFactory"
    )

    class Meta:
        """
        Metaclass for RateFactory
        """

        model = "dispatch.Rate"
        django_get_or_create = (
            "organization",
            "commodity",
            "order_type",
            "equipment_type",
        )
