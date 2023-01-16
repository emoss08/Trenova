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
from factory.fuzzy import FuzzyDecimal

from order import models


class OrderTypeFactory(factory.django.DjangoModelFactory):
    """
    OrderType factory
    """

    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    name = factory.Faker("word", locale="en_US")

    class Meta:
        """
        Metaclass for OrderTypeFactory
        """

        model = models.OrderType


class ReasonCodeFactory(factory.django.DjangoModelFactory):
    """
    ReasonCode Factory
    """

    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    code = factory.Faker("pystr", max_chars=4)
    description = factory.Faker("text", locale="en_US", max_nb_chars=100)
    code_type = factory.Faker(
        "random_element",
        elements=("VOIDED", "CANCELLED"),
    )

    class Meta:
        """
        Metaclass for ReasonCodeFactory
        """

        model = models.ReasonCode


class OrderFactory(factory.django.DjangoModelFactory):
    """
    Order Factory
    """

    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    order_type = factory.SubFactory(OrderTypeFactory)
    status = "N"
    revenue_code = factory.SubFactory("accounting.tests.factories.RevenueCodeFactory")
    origin_location = factory.SubFactory("location.factories.LocationFactory")
    origin_appointment = factory.Faker("date_time", locale="en_US")
    destination_location = factory.SubFactory("location.factories.LocationFactory")
    destination_appointment = factory.Faker("date_time", locale="en_US")
    customer = factory.SubFactory("customer.factories.CustomerFactory")
    equipment_type = factory.SubFactory(
        "equipment.tests.factories.EquipmentTypeFactory"
    )
    bol_number = factory.Faker("text", locale="en_US", max_nb_chars=100)
    entered_by = factory.SubFactory("accounts.tests.factories.UserFactory")

    class Meta:
        """
        Metaclass for orderFactory
        """

        model = models.Order


class OrderCommentFactory(factory.django.DjangoModelFactory):
    """
    Order Comment Factory
    """

    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    order = factory.SubFactory(OrderFactory)
    comment_type = factory.SubFactory("dispatch.factories.CommentTypeFactory")
    comment = factory.Faker("text", locale="en_US", max_nb_chars=100)
    entered_by = factory.SubFactory("accounts.tests.factories.UserFactory")

    class Meta:
        """
        Metaclass For OrderCommentFactory
        """

        model = models.OrderComment


class OrderDocumentationFactory(factory.django.DjangoModelFactory):
    """
    Order Documentation Factory
    """

    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    order = factory.SubFactory(OrderFactory)
    document = factory.django.FileField(filename="test.txt")
    document_class = factory.SubFactory(
        "billing.tests.factories.DocumentClassificationFactory"
    )

    class Meta:
        """
        Metaclass for OrderDocumentationFactory
        """

        model = models.OrderDocumentation


class AdditionalChargeFactory(factory.django.DjangoModelFactory):
    """
    AdditionalCharge Factory
    """

    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    order = factory.SubFactory(OrderFactory)
    charge = factory.SubFactory("billing.tests.factories.AccessorialChargeFactory")
    charge_amount = FuzzyDecimal(low=10.00, high=100000.00, precision=2)
    sub_total = FuzzyDecimal(low=10.00, high=100000.00, precision=2)
    entered_by = factory.SubFactory("accounts.tests.factories.UserFactory")

    class Meta:
        """
        Metaclass for AdditionalChargeFactory
        """

        model = models.AdditionalCharge
