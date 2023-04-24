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

from integration import exceptions, models
from organization.models import Organization


def get_maps_api_key(*, organization: Organization) -> str:
    """Get the Google Maps API key for the given organization.

    Args:
        organization (Organization): The organization to get the Google Maps API key for.

    Returns:
        str: The Google Maps API key for the given organization.

    Raises:
        ValueError: If the GoogleAPI object for the given organization does not exist.
    """
    try:
        return models.GoogleAPI.objects.get(organization=organization).api_key
    except models.GoogleAPI.DoesNotExist as google_api_exception:
        raise exceptions.GoogleAPINotFoundError(
            organization.name
        ) from google_api_exception


def get_organization_google_api(*, organization: Organization) -> str | None:
    try:
        return models.GoogleAPI.objects.get(organization=organization)
    except models.GoogleAPI.DoesNotExist:
        return
