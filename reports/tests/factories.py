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


class CustomReportFactory(factory.django.DjangoModelFactory):
    """
    Custom Report Factory
    """

    class Meta:
        """
        Metaclass for CustomReportFactory
        """

        model = "reports.CustomReport"
        django_get_or_create = ("organization",)

    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    name = factory.Faker("word", locale="en_US")
    table = factory.Faker(
        "random_element",
        elements=(
            "organization",
            "depot",
            "email_profile",
        ),
    )


class ReportColumnFactory(factory.django.DjangoModelFactory):
    """
    Report Column Factory
    """

    class Meta:
        """
        Metaclass for ReportColumnFactory
        """

        model = "reports.ReportColumn"
        django_get_or_create = ("organization", "report")

    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    report = factory.SubFactory(CustomReportFactory)
    column_name = factory.Faker("word", locale="en_US")
    column_order = 1


class ScheduledReportFactory(factory.django.DjangoModelFactory):
    """
    Scheduled Report Factory
    """

    class Meta:
        """
        Metaclass for ScheduledReportFactory
        """

        model = "reports.ScheduledReport"
        django_get_or_create = ("organization", "report")

    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    report = factory.SubFactory(CustomReportFactory)
    user = factory.SubFactory("accounts.test.factories.UserFactory")
    schedule_type = "WEEKLY"
    time = "12:00"
    day_of_week = 0  # monday
    timezone = "UTC"
