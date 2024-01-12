# --------------------------------------------------------------------------------------------------
#  COPYRIGHT(c) 2024 Trenova                                                                       -
#                                                                                                  -
#  This file is part of Trenova.                                                                   -
#                                                                                                  -
#  The Trenova software is licensed under the Business Source License 1.1. You are granted the right
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
from django.core.exceptions import ValidationError

from organization.models import Organization

pytestmark = pytest.mark.django_db


def test_trenova_distance_method_with_route_generation_error(
    organization: Organization,
) -> None:
    """Test ValidationError is thrown when Trenova is selected as the distance method and route generation is enabled.

    Args:
        organization: Organization Object.

    Returns:
        None: This function does not return anything.
    """
    route_control = organization.route_control
    route_control.distance_method = "M"
    route_control.generate_routes = True

    with pytest.raises(ValidationError) as excinfo:
        route_control.full_clean()

    assert excinfo.value.message_dict == {
        "generate_routes": [
            "'Trenova' does not support automatic route generation. Please select Google as the distance method."
        ]
    }


def test_google_distance_method_without_integration_error(
    organization: Organization,
) -> None:
    """Test ValidationError is thrown when google is set as the distance method,
    but an integration does not exist within the organization for google.

    Args:
        organization(Organization): Organization Object.

    Returns:
        None: This function does not return anything.
    """

    route_control = organization.route_control
    route_control.distance_method = "G"

    with pytest.raises(ValidationError) as excinfo:
        route_control.full_clean()

    assert excinfo.value.message_dict == {
        "distance_method": [
            "Google Maps integration is not configured for the organization. Please configure the integration before "
            "selecting Google as the distance method."
        ]
    }
