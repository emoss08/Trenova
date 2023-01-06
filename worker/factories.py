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


class WorkerFactory(factory.django.DjangoModelFactory):
    """
    Worker factory
    """

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

    class Meta:
        """
        Metaclass for WorkerFactory
        """

        model = "worker.Worker"

    @factory.post_generation
    def worker_contact(self, create, extracted, **kwargs):
        """
        WorkerContact post generation
        """
        if not create:
            return

        if extracted:
            self.worker_contact = extracted
        else:
            self.worker_contact = WorkerContactFactory(worker=self)

    @factory.post_generation
    def worker_comment(self, create, extracted, **kwargs):
        """
        WorkerComment post generation
        """
        if not create:
            return

        if extracted:
            self.worker_comment = extracted
        else:
            self.worker_comment = WorkerCommentFactory(worker=self)


class WorkerContactFactory(factory.django.DjangoModelFactory):
    """
    WorkerContact factory
    """

    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    worker = factory.SubFactory("worker.factories.WorkerFactory")
    name = factory.Faker("name", locale="en_US")
    email = factory.Faker("email", locale="en_US")

    class Meta:
        """
        Metaclass for WorkerContactFactory
        """

        model = "worker.WorkerContact"


class WorkerCommentFactory(factory.django.DjangoModelFactory):
    """
    WorkerComment factory
    """

    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    worker = factory.SubFactory("worker.factories.WorkerFactory")
    comment_type = factory.SubFactory("dispatch.factories.CommentTypeFactory")
    comment = factory.Faker("text", locale="en_US", max_nb_chars=100)
    entered_by = factory.SubFactory("accounts.tests.factories.UserFactory")

    class Meta:
        """
        Metaclass for WorkerCommentFactory
        """

        model = "worker.WorkerComment"
