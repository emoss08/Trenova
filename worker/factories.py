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


class WorkerFactory(factory.django.DjangoModelFactory):
    """
    Worker factory
    """

    class Meta:
        """
        Metaclass for WorkerFactory
        """

        model = "worker.Worker"
        django_get_or_create = ("organization",)

    business_unit = factory.SubFactory("organization.factories.BusinessUnitFactory")
    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    code = factory.Faker("text", locale="en_US", max_nb_chars=10)
    first_name = factory.Faker("name")
    last_name = factory.Faker("name")
    worker_type = "EMPLOYEE"
    address_line_1 = factory.Faker("street_address")
    address_line_2 = factory.Faker("secondary_address")
    city = factory.Faker("city")
    state = "CA"
    zip_code = factory.Faker("zipcode")
    manager = factory.SubFactory("accounts.tests.factories.UserFactory")
    entered_by = factory.SubFactory("accounts.tests.factories.UserFactory")
    fleet = factory.SubFactory("dispatch.factories.FleetCodeFactory")

    @factory.post_generation
    def worker_contact(self, create, extracted, **kwargs):
        """
        WorkerContact post generation
        """
        if not create:
            return

        self.worker_contact = extracted or WorkerContactFactory(worker=self)

    @factory.post_generation
    def worker_comment(self, create, extracted, **kwargs):
        """
        WorkerComment post generation
        """
        if not create:
            return

        self.worker_comment = extracted or WorkerCommentFactory(worker=self)


class WorkerContactFactory(factory.django.DjangoModelFactory):
    """
    WorkerContact factory
    """

    class Meta:
        """
        Metaclass for WorkerContactFactory
        """

        model = "worker.WorkerContact"
        django_get_or_create = (
            "organization",
            "worker",
        )

    business_unit = factory.SubFactory("organization.factories.BusinessUnitFactory")
    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    worker = factory.SubFactory("worker.factories.WorkerFactory")
    name = factory.Faker("name", locale="en_US")
    email = factory.Faker("email", locale="en_US")


class WorkerCommentFactory(factory.django.DjangoModelFactory):
    """
    WorkerComment factory
    """

    class Meta:
        """
        Metaclass for WorkerCommentFactory
        """

        model = "worker.WorkerComment"
        django_get_or_create = (
            "organization",
            "worker",
            "comment_type",
            "entered_by",
        )

    business_unit = factory.SubFactory("organization.factories.BusinessUnitFactory")
    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    worker = factory.SubFactory("worker.factories.WorkerFactory")
    comment_type = factory.SubFactory("dispatch.factories.CommentTypeFactory")
    comment = factory.Faker("text", locale="en_US", max_nb_chars=100)
    entered_by = factory.SubFactory("accounts.tests.factories.UserFactory")
