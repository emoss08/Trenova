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


class LocationCategoryFactory(factory.django.DjangoModelFactory):
    """
    LocationCategory factory
    """

    class Meta:
        """
        Metaclass for LocationCategoryFactory
        """

        model = "location.LocationCategory"

    business_unit = factory.SubFactory("organization.factories.BusinessUnitFactory")
    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    name = factory.Faker("pystr", max_chars=100)


class LocationFactory(factory.django.DjangoModelFactory):
    """
    Location factory
    """

    class Meta:
        """
        Metaclass for LocationFactory
        """

        model = "location.Location"

    business_unit = factory.SubFactory("organization.factories.BusinessUnitFactory")
    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    code = factory.Faker("pystr", max_chars=100)
    location_category = factory.SubFactory("location.factories.LocationCategoryFactory")
    address_line_1 = factory.Faker("address", locale="en_US")
    city = factory.Faker("city", locale="en_US")
    state = "NC"
    zip_code = factory.Faker("zipcode", locale="en_US")


class LocationContactFactory(factory.django.DjangoModelFactory):
    """
    LocationContact factory
    """

    class Meta:
        """
        Metaclaszs for LocationContactFactory
        """

        model = "location.LocationContact"

    business_unit = factory.SubFactory("organization.factories.BusinessUnitFactory")
    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    location = factory.SubFactory("location.factories.LocationFactory")
    name = factory.Faker("pystr", max_chars=100)
    email = factory.Faker("email", locale="en_US")


class LocationCommentFactory(factory.django.DjangoModelFactory):
    """
    LocationComment factory
    """

    class Meta:
        """
        Metaclass for LocationCommentFactory
        """

        model = "location.LocationComment"

    business_unit = factory.SubFactory("organization.factories.BusinessUnitFactory")
    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    location = factory.SubFactory("location.factories.LocationFactory")
    comment_type = factory.SubFactory("dispatch.factories.CommentTypeFactory")
    comment = factory.Faker("text", locale="en_US")
    entered_by = factory.SubFactory("accounts.tests.factories.UserFactory")
