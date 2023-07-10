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

    business_unit = factory.SubFactory("organization.factories.BusinessUnitFactory")
    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    status = "N"
    sequence = 1
    movement = factory.SubFactory("movements.tests.factories.MovementFactory")
    location = factory.SubFactory("location.factories.LocationFactory")
    pieces = factory.Faker("pyint", min_value=1, max_value=100)
    weight = factory.Faker("pyint", min_value=1, max_value=100)
    address_line = factory.Faker("street_address", locale="en_US")
    appointment_time_window_start = factory.Faker(
        "date_time", tzinfo=timezone.get_current_timezone()
    )
    appointment_time_window_end = factory.Faker(
        "date_time", tzinfo=timezone.get_current_timezone()
    )
    stop_type = "P"
