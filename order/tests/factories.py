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
from django.core.files.uploadedfile import SimpleUploadedFile
from django.utils import timezone
from factory.fuzzy import FuzzyDecimal

from utils.models import RatingMethodChoices


class OrderTypeFactory(factory.django.DjangoModelFactory):
    """
    OrderType factory
    """

    class Meta:
        """
        Metaclass for OrderTypeFactory
        """

        model = "order.OrderType"
        django_get_or_create = ("organization",)

    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    name = factory.Faker("pystr", max_chars=255)


class ReasonCodeFactory(factory.django.DjangoModelFactory):
    """
    ReasonCode Factory
    """

    class Meta:
        """
        Metaclass for ReasonCodeFactory
        """

        model = "order.ReasonCode"
        django_get_or_create = ("organization",)

    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    code = factory.Faker("pystr", max_chars=4)
    description = factory.Faker("text", locale="en_US", max_nb_chars=100)
    code_type = factory.Faker(
        "random_element",
        elements=("VOIDED", "CANCELLED"),
    )


class OrderFactory(factory.django.DjangoModelFactory):
    """
    Order Factory
    """

    class Meta:
        """
        Metaclass for orderFactory
        """

        model = "order.Order"
        django_get_or_create = (
            "organization",
            "order_type",
            "revenue_code",
            "origin_location",
            "destination_location",
            "customer",
        )

    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    order_type = factory.SubFactory(OrderTypeFactory)
    status = "N"
    revenue_code = factory.SubFactory("accounting.tests.factories.RevenueCodeFactory")
    origin_location = factory.SubFactory("location.factories.LocationFactory")
    origin_appointment_window_start = factory.Faker(
        "date_time", tzinfo=timezone.get_current_timezone()
    )
    origin_appointment_window_end = factory.Faker(
        "date_time", tzinfo=timezone.get_current_timezone()
    )
    destination_location = factory.SubFactory("location.factories.LocationFactory")
    rate_method = RatingMethodChoices.FLAT
    freight_charge_amount = FuzzyDecimal(10, 1000000, 4)
    destination_appointment_window_start = factory.Faker(
        "date_time", tzinfo=timezone.get_current_timezone()
    )
    destination_appointment_window_end = factory.Faker(
        "date_time", tzinfo=timezone.get_current_timezone()
    )
    customer = factory.SubFactory("customer.factories.CustomerFactory")
    equipment_type = factory.SubFactory(
        "equipment.tests.factories.EquipmentTypeFactory"
    )
    bol_number = factory.Faker("text", locale="en_US", max_nb_chars=100)
    entered_by = factory.SubFactory("accounts.tests.factories.UserFactory")


class OrderCommentFactory(factory.django.DjangoModelFactory):
    """
    Order Comment Factory
    """

    class Meta:
        """
        Metaclass For OrderCommentFactory
        """

        model = "order.OrderComment"
        django_get_or_create = ("organization",)

    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    order = factory.SubFactory(OrderFactory)
    comment_type = factory.SubFactory("dispatch.factories.CommentTypeFactory")
    comment = factory.Faker("text", locale="en_US", max_nb_chars=100)
    entered_by = factory.SubFactory("accounts.tests.factories.UserFactory")


class OrderDocumentationFactory(factory.django.DjangoModelFactory):
    """
    Order Documentation Factory
    """

    class Meta:
        """
        Metaclass for OrderDocumentationFactory
        """

        model = "order.OrderDocumentation"
        django_get_or_create = ("organization", "order", "document_class")

    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    order = factory.SubFactory(OrderFactory)
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

        model = "order.AdditionalCharge"
        django_get_or_create = (
            "organization",
            "order",
            "accessorial_charge",
            "entered_by",
        )

    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    order = factory.SubFactory(OrderFactory)
    accessorial_charge = factory.SubFactory(
        "billing.tests.factories.AccessorialChargeFactory"
    )
    charge_amount = FuzzyDecimal(low=10.00, high=100000.00, precision=4)
    sub_total = FuzzyDecimal(low=10.00, high=100000.00, precision=4)
    entered_by = factory.SubFactory("accounts.tests.factories.UserFactory")
