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
from typing import Any, Optional, Tuple, Union

import googlemaps

from integration import selectors
from location.models import Location
from organization.models import Organization


def google_client(*, organization: Organization) -> googlemaps.Client:
    api_key = selectors.get_maps_api_key(organization=organization)
    return googlemaps.Client(key=api_key)


def geocode_location_service(*, location: Location) -> None:
    gmaps_api_config = selectors.get_organization_google_api(
        organization=location.organization
    )

    if not gmaps_api_config:
        return

    gmaps = google_client(organization=location.organization)
    if geocode_result := gmaps.geocode(
        f"{location.address_line_1}, {location.city}, {location.state} {location.zip_code}"
    ):
        location.latitude = geocode_result[0]["geometry"]["location"]["lat"]
        location.longitude = geocode_result[0]["geometry"]["location"]["lng"]
        location.place_id = geocode_result[0]["place_id"]
        location.is_geocoded = True


def google_distance_matrix_service(
    *,
    point_1: tuple[float | None, float | None] | Any,
    point_2: tuple[float | None, float | None] | Any,
    units: str,
    organization: Organization,
) -> tuple[float | Any, float | Any]:
    gmaps = google_client(organization=organization)
    distance_matrix = gmaps.distance_matrix(
        origins=point_1,
        destinations=point_2,
        mode="driving",
        units=f"{units}",
    )
    distance_meters = distance_matrix["rows"][0]["elements"][0]["distance"]["value"]
    distance_miles = distance_meters * 0.000621371
    duration_seconds = distance_matrix["rows"][0]["elements"][0]["duration"]["value"]
    duration_hours = duration_seconds / 3600
    return round(distance_miles, 2), round(duration_hours, 2)
