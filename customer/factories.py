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

from customer import models


class CustomerFactory(factory.django.DjangoModelFactory):
    """
    Customer factory
    """

    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    name = factory.Faker("name", locale="en_US")

    class Meta:
        """
        Metaclass for CustomerFactory
        """

        model = models.Customer


class CustomerContactFactory(factory.django.DjangoModelFactory):
    """
    Customer contact factory
    """

    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    customer = factory.SubFactory(CustomerFactory)
    name = factory.Faker("name", locale="en_US")
    email = factory.Faker("email", locale="en_US")
    title = factory.Faker("word", locale="en_US")
    is_payable_contact = True

    class Meta:
        """
        Metaclass for CustomerContactFactory
        """

        model = models.CustomerContact
