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
from typing import Any, Literal

import googlemaps

from integration import selectors
from location.models import Location
from organization.models import Organization
from utils.types import Point


def google_client(*, organization: Organization) -> googlemaps.Client:
    """Initializes a Google Maps Client using the API key associated with the provided organization.

    Args:
        organization (Organization): The organization for which the Google Maps client will be configured.

    Returns:
        googlemaps.Client: An instance of the Google Maps client configured with the organization's API key.

    Raises:
        googlemaps.exceptions.ApiError: If the API key is invalid or quota is exceeded.
    """

    api_key = selectors.get_maps_api_key(organization=organization)
    return googlemaps.Client(key=api_key)


def geocode_location_service(*, location: Location) -> None:
    """Fetches and stores the latitude, longitude, and place ID for a given location by querying the Google Maps Geocoding API.

    Args:
        location (Location): The location instance containing address details to geocode.

    Updates:
        Updates the provided location instance's latitude, longitude, place_id, and is_geocoded fields if geocoding is successful.

    Notes:
        The function does not return any value. It directly modifies the passed location instance.
        If the organization associated with the location does not have a Google API configured or geocoding fails, no changes are made.
    """
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
    point_1: Point,
    point_2: Point,
    units: Literal["metric", "imperial"] = "imperial",
    organization: Organization,
) -> tuple[float | Any, float | Any]:
    """Calculates the driving distance and duration between two points using the Google Maps Distance Matrix API.

    Args:
        point_1 (Point): The origin point (latitude, longitude).
        point_2 (Point): The destination point (latitude, longitude).
        units (str): The unit system to use for results (e.g., 'metric' or 'imperial').
        organization (Organization): The organization whose API key will be used for the Google Maps client.

    Returns:
        tuple[float | Any, float | Any]: A tuple containing the distance in miles (rounded to 2 decimal places) and duration in hours (rounded to 2 decimal places).

    Notes:
        If the API request fails or does not return valid results, the function will return (None, None).
    """
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


def autocomplete_location_service(
    *, organization: Organization, search_query: str
) -> list[dict[str, Any]]:
    defaults = {
        "input_text": search_query,
        "language": "en-US",
    }

    gmaps = google_client(organization=organization)
    autocomplete_results = gmaps.places_autocomplete(**defaults)

    # Process the results and fetch details
    detailed_results = []
    for result in autocomplete_results:
        place_id = result.get("place_id", "")

        # Fetch detailed information for each place
        detailed_info = gmaps.place(place_id=place_id)  # Corrected this line

        # Extract and structure address components
        address_components = {
            component["types"][0]: component["long_name"]
            for component in detailed_info["result"].get("address_components", [])
        }
        formatted_address = detailed_info["result"].get("formatted_address", "")

        place_details = {
            "name": result.get("structured_formatting", {}).get("main_text", ""),
            "address": formatted_address,
            "address_components": address_components,
            "place_id": place_id,
        }
        detailed_results.append(place_details)

    return detailed_results
