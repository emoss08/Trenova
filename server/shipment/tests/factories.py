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
from datetime import timedelta

import factory
from django.core.files.uploadedfile import SimpleUploadedFile
from django.utils import timezone
from factory.fuzzy import FuzzyDecimal

from utils.models import RatingMethodChoices


class ServiceTypeFactory(factory.django.DjangoModelFactory):
    """
    ServiceType Factory
    """

    class Meta:
        """
        Metaclass for ServiceTypeFactory
        """

        model = "shipment.ServiceType"

    business_unit = factory.SubFactory("organization.factories.BusinessUnitFactory")
    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    code = factory.Faker("pystr", max_chars=4)
    description = factory.Faker("text", locale="en_US", max_nb_chars=100)


class ShipmentTypeFactory(factory.django.DjangoModelFactory):
    """
    ShipmentType factory
    """

    class Meta:
        """
        Metaclass for ShipmentTypeFactory
        """

        model = "shipment.ShipmentType"
        django_get_or_create = ("code",)

    business_unit = factory.SubFactory("organization.factories.BusinessUnitFactory")
    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    code = factory.Faker("pystr", max_chars=10)


class ReasonCodeFactory(factory.django.DjangoModelFactory):
    """
    ReasonCode Factory
    """

    class Meta:
        """
        Metaclass for ReasonCodeFactory
        """

        model = "shipment.ReasonCode"
        django_get_or_create = ("code",)

    business_unit = factory.SubFactory("organization.factories.BusinessUnitFactory")
    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    code = factory.Faker("pystr", max_chars=4)
    description = factory.Faker("text", locale="en_US", max_nb_chars=100)
    code_type = factory.Faker(
        "random_element",
        elements=("VOIDED", "CANCELLED"),
    )


class ShipmentFactory(factory.django.DjangoModelFactory):
    """
    shipment Factory
    """

    class Meta:
        """
        Metaclass for ShipmentFactory
        """

        model = "shipment.Shipment"

    business_unit = factory.SubFactory("organization.factories.BusinessUnitFactory")
    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    shipment_type = factory.SubFactory(ShipmentTypeFactory)
    status = "N"
    revenue_code = factory.SubFactory("accounting.tests.factories.RevenueCodeFactory")
    origin_location = factory.SubFactory("location.factories.LocationFactory")
    origin_appointment_window_start = timezone.now()
    origin_appointment_window_end = timezone.now()
    destination_location = factory.SubFactory("location.factories.LocationFactory")
    service_type = factory.SubFactory(ServiceTypeFactory)
    rate_method = RatingMethodChoices.FLAT
    freight_charge_amount = FuzzyDecimal(10, 1000000, 4)
    destination_appointment_window_start = timezone.now() + timedelta(days=1)
    destination_appointment_window_end = timezone.now() + timedelta(days=1)
    customer = factory.SubFactory("customer.factories.CustomerFactory")
    equipment_type = factory.SubFactory(
        "equipment.tests.factories.EquipmentTypeFactory"
    )
    bol_number = factory.Faker("text", locale="en_US", max_nb_chars=100)
    entered_by = factory.SubFactory("accounts.tests.factories.UserFactory")
    pieces = 1


class ShipmentCommentFactory(factory.django.DjangoModelFactory):
    """
    shipment Comment Factory
    """

    class Meta:
        """
        Metaclass For ShipmentCommentFactory
        """

        model = "shipment.ShipmentComment"

    business_unit = factory.SubFactory("organization.factories.BusinessUnitFactory")
    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    shipment = factory.SubFactory(ShipmentFactory)
    comment_type = factory.SubFactory("dispatch.factories.CommentTypeFactory")
    comment = factory.Faker("text", locale="en_US", max_nb_chars=100)
    entered_by = factory.SubFactory("accounts.tests.factories.UserFactory")


class ShipmentDocumentationFactory(factory.django.DjangoModelFactory):
    """
    shipment Documentation Factory
    """

    class Meta:
        """
        Metaclass for ShipmentDocumentationFactory
        """

        model = "shipment.ShipmentDocumentation"

    business_unit = factory.SubFactory("organization.factories.BusinessUnitFactory")
    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    shipment = factory.SubFactory(ShipmentFactory)
    document = SimpleUploadedFile(
        "file.pdf", b"file_content", content_type="application/pdf"
    )
    document_class = factory.SubFactory(
        "billing.tests.factories.DocumentClassificationFactory"
    )


class AdditionalChargeFactory(factory.django.DjangoModelFactory):
    """
    AdditionalCharge Factory
    """

    class Meta:
        """
        Metaclass for AdditionalChargeFactory
        """

        model = "shipment.AdditionalCharge"

    business_unit = factory.SubFactory("organization.factories.BusinessUnitFactory")
    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    shipment = factory.SubFactory(ShipmentFactory)
    accessorial_charge = factory.SubFactory(
        "billing.tests.factories.AccessorialChargeFactory"
    )
    charge_amount = FuzzyDecimal(low=10.00, high=100000.00, precision=4)
    sub_total = FuzzyDecimal(low=10.00, high=100000.00, precision=4)
    entered_by = factory.SubFactory("accounts.tests.factories.UserFactory")
