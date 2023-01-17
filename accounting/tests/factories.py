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


class GeneralLedgerAccountFactory(factory.django.DjangoModelFactory):
    """
    General Ledger Account factory
    """
    class Meta:
        """
        Metaclass for GeneralLedger Account Factory
        """
        model = "accounting.GeneralLedgerAccount"
        django_get_or_create = ("organization",)

    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    account_number = factory.Faker(
        "numerify",
        text="####-####-####-####",
    )
    account_type = factory.Faker(
        "random_element",
        elements=("ASSET", "LIABILITY", "EQUITY", "REVENUE", "EXPENSE"),
    )
    description = factory.Faker("sentence", nb_words=6, locale="en_US")



class RevenueCodeFactory(factory.django.DjangoModelFactory):
    """
    Revenue Code factory
    """
    class Meta:
        """
        Metaclass for RevenueCode Factory
        """
        model = "accounting.RevenueCode"
        django_get_or_create = ("organization",)

    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    code = factory.Faker("pystr", max_chars=4)
    description = factory.Faker("text", locale="en_US", max_nb_chars=100)


class DivisionCodeFactory(factory.django.DjangoModelFactory):
    """
    Division Code Factory
    """
    class Meta:
        """
        Metaclass for DivisionCodeFactory
        """
        model = "accounting.DivisionCode"
        django_get_or_create = ("organization",)

    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    is_active = True
    code = factory.Faker("pystr", max_chars=4)
    description = factory.Faker("text", locale="en_US", max_nb_chars=100)
