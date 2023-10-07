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

    business_unit = factory.SubFactory("organization.factories.BusinessUnitFactory")
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

    business_unit = factory.SubFactory("organization.factories.BusinessUnitFactory")
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

    business_unit = factory.SubFactory("organization.factories.BusinessUnitFactory")
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

    business_unit = factory.SubFactory("organization.factories.BusinessUnitFactory")
    organization = factory.SubFactory("organization.factories.OrganizationFactory")
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

    business_unit = factory.SubFactory("organization.factories.BusinessUnitFactory")
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


class DeliverySlotFactory(factory.django.DjangoModelFactory):
    """
    DeliverySlot factory
    """

    class Meta:
        """
        Metaclass for DeliverySlot
        """

        model = "customer.DeliverySlot"

    business_unit = factory.SubFactory("organization.factories.BusinessUnitFactory")
    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    customer = factory.SubFactory(CustomerFactory)
    day_of_week = "WED"
    start_time = "00:00:00"
    end_time = "23:59:59"
    location = factory.SubFactory("location.factories.LocationFactory")
