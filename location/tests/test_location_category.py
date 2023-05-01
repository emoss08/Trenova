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
def location_category():
    """
    Location category fixture
    """
    return factories.LocationCategoryFactory()


def test_location_category_creation(location_category: models.LocationComment) -> None:
    """
    Test location category creation
    """
    assert location_category is not None


def test_location_category_update(location_category: models.LocationComment) -> None:
    """
    Test location category update
    """
    location_category.name = "New name"
    location_category.save()
    assert location_category.name == "New name"


def test_add_category_to_location(location_category: models.LocationComment) -> None:
    """
    Test add category to location
    """
    location = factories.LocationCategoryFactory()
    location.location_category = location_category
    location.save()
    assert location.location_category == location_category
