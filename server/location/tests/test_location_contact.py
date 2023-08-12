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

import pytest
from location import factories, models

pytestmark = pytest.mark.django_db


@pytest.fixture()
def location_contact():
    """
    Location fixture
    """
    return factories.LocationContactFactory()


def test_add_contact_to_location(location_contact: models.LocationContact):
    """
    Test add contact to location
    """
    location = factories.LocationContactFactory()
    location.location_contact = location_contact
    location.save()
    assert location.location_contact == location_contact


def test_update_contact_on_location(location_contact: models.LocationContact) -> None:
    """
    Test update contact to location
    """
    location = factories.LocationContactFactory()
    location.location_contact = location_contact
    location.save()
    location.location_contact.name = "New name"
    location.save()
    assert location.location_contact.name == "New name"
