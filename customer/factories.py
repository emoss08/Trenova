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
