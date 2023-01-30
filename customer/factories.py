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


class CustomerFactory(factory.django.DjangoModelFactory):
    """
    Customer factory
    """

    class Meta:
        """
        Metaclass for CustomerFactory
        """

        model = "customer.Customer"
        django_get_or_create = ("organization",)

    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    name = factory.Faker("name", locale="en_US")


class DocumentClassificationFactory(factory.django.DjangoModelFactory):
    """
    Document Classification factory
    """

    class Meta:
        """
        Metaclass for DocumentClassificationFactory
        """

        model = "billing.DocumentClassification"
        django_get_or_create = ("organization",)

    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    name = factory.Faker("name", locale="en_US")


class CustomerContactFactory(factory.django.DjangoModelFactory):
    """
    Customer contact factory
    """

    class Meta:
        """
        Metaclass for CustomerContactFactory
        """

        model = "customer.CustomerContact"
        django_get_or_create = ("organization", "customer")

    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    customer = factory.SubFactory(CustomerFactory)
    name = factory.Faker("name", locale="en_US")
    email = factory.Faker("email", locale="en_US")
    title = factory.Faker("word", locale="en_US")
    is_payable_contact = True


class CustomerEmailProfileFactory(factory.django.DjangoModelFactory):
    """
    Customer Email Profile Factory
    """

    class Meta:
        """
        Metaclass for CustomerEmailProfileFactory
        """

        model = "customer.CustomerEmailProfile"
        django_get_or_create = ("organization",)

    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    name = factory.Faker("word", locale="en_US")
    subject = factory.Faker("word", locale="en_US")
    comment = factory.Faker("word", locale="en_US")
    from_address = factory.Faker("email", locale="en_US")
    blind_copy = factory.Faker("email", locale="en_US")
    read_receipt = False
    attachment_name = factory.Faker("word", locale="en_US")


class CustomerRuleProfileFactory(factory.django.DjangoModelFactory):
    """
    Customer rule profile factory
    """

    class Meta:
        """
        Metaclass for CustomerRuleProfileFactory
        """

        model = "customer.CustomerRuleProfile"
        django_get_or_create = ("organization",)

    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    name = factory.Faker("word", locale="en_US")

    @factory.post_generation
    def document_class(self, create, extracted, **kwargs):
        """
        Post generation method for document classes
        """

        if not create:
            return

        if extracted:
            for document_class in extracted:
                self.document_class.add(document_class)


class CustomerBillingProfileFactory(factory.django.DjangoModelFactory):
    """
    Customer Billing Profile factory
    """

    class Meta:
        """
        Metaclass for CustomerBillingFactory
        """

        model = "customer.CustomerBillingProfile"
        django_get_or_create = (
            "organization",
            "customer",
            "email_profile",
            "rule_profile",
        )

    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    customer = factory.SubFactory(CustomerFactory)
    email_profile = factory.SubFactory(CustomerEmailProfileFactory)
    rule_profile = factory.SubFactory(CustomerRuleProfileFactory)
