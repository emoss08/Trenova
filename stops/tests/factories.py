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
from django.utils import timezone


class StopFactory(factory.django.DjangoModelFactory):
    """
    Stop Factory
    """

    class Meta:
        """
        Metaclass for StopFactory
        """

        model = "stops.Stop"
        django_get_or_create = (
            "organization",
            "movement",
            "location",
        )

    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    status = "N"
    sequence = 1
    movement = factory.SubFactory("movements.tests.factories.MovementFactory")
    location = factory.SubFactory("location.factories.LocationFactory")
    pieces = factory.Faker("pyint", min_value=1, max_value=100)
    weight = factory.Faker("pyint", min_value=1, max_value=100)
    address_line = factory.Faker("street_address", locale="en_US")
    appointment_time = factory.Faker(
        "date_time", tzinfo=timezone.get_current_timezone()
    )
    stop_type = "P"

