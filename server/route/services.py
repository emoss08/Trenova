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

from typing import Any

from geopy.distance import geodesic

from integration.services import google_distance_matrix_service
from route import models
from route.models import RouteControl
from shipment.models import Shipment
from utils.types import Coordinates


def generate_route(
    *, shipment: Shipment, distance: float, method: str, duration: float | None
) -> None:
    """Generate a new Route object for a Shipment.

    This function takes an instance of the Shipment model representing an Shipment, as well as the distance,
    method, and duration of the new Route object, and creates a new Route object in the database using
    the organization, origin_location, destination_location, total_mileage, duration, and distance_method attributes.

    Args:
        shipment: An instance of the Shipment model representing a shipment to generate a new Route object for.
        distance: A float representing the total distance of the new Route object in miles.
        method: A string representing the method used to calculate the distance ('Google' or 'Monta').
        duration: A string representing the duration of the new Route object in hours, or None if the method is 'Monta'.

    Returns:
        None: This function does not return anything.
    """

    # Create a new Route object in the database or update an existing Route object
    models.Route.objects.update_or_create(
        organization=shipment.organization,
        origin_location=shipment.origin_location,
        destination_location=shipment.destination_location,
        total_mileage=distance,
        duration=duration,
        distance_method=method,
        auto_generated=True,
    )


def get_coordinates(*, shipment: Shipment) -> Coordinates:
    """
    Retrieve the latitude and longitude coordinates for a Shipment's origin and destination locations.

    This function takes an instance of the Shipment model representing a Shipment and retrieves the latitude
    and longitude coordinates for the Shipment's origin and destination locations. The function returns a
    tuple containing two tuples, each representing a pair of latitude and longitude coordinates for the two points.

    Args:
        shipment: An instance of the Shipment model representing a shipment to retrieve the coordinates for.

    Returns:
        A tuple containing two tuples, each representing a pair of latitude and longitude coordinates for
        the Shipment's origin and destination locations.
    """

    # Return None if the Shipment does not have an origin or destination location
    if not shipment.origin_location or not shipment.destination_location:
        return None

    # Get the latitude and longitude coordinates for the shipment's origin and destination locations
    point_1 = (shipment.origin_location.latitude, shipment.origin_location.longitude)
    point_2 = (
        shipment.destination_location.latitude,
        shipment.destination_location.longitude,
    )

    # Return the coordinates
    return point_1, point_2


def calculate_distance(
    *,
    route_control: RouteControl,
    point_1: tuple[float | None, float | None] | Any,
    point_2: tuple[float | None, float | None] | Any,
) -> tuple[float, float, str]:
    """
    Calculate the distance and duration between two points on the Earth's surface using the Haversine formula or the
    Google Distance Matrix API.

    This function takes an instance of the Organization model representing the organization associated with the Shipment,
    as well as two tuples representing the latitude and longitude coordinates of the two points to calculate the distance
    between. The function first retrieves the organization's route_control attribute to determine which method to use
    for calculating the distance. If the distance_method is set to GOOGLE, the function uses the Google Distance Matrix
    API to retrieve the distance and duration between the two points. Otherwise, the function calculates the distance
    using the geodesic() function from the geopy library, which implements the Haversine formula.

    Args:
        route_control (RouteControl): An instance of the Organization's RouteControl object.
        point_1 (Union[Tuple[Optional[Float], Optional[Float]], Any]): A tuple of two floats representing the latitude
        and longitude of the first point.
        point_2 (Union[Tuple[Optional[Float], Optional[Float]], Any]): A tuple of two floats representing the latitude
        and longitude of the second point.

    Returns:
        A tuple containing three values: a float representing the distance between the two points in miles, a string
        representing the method used to calculate the distance ('Google' or 'Monta'), and a float representing the
        duration between the two points in hours if the method is 'Google', or None if the method is 'Monta'.
    """

    # Get the distance method from the Organization's RouteControl object
    if route_control.distance_method == RouteControl.DistanceMethodChoices.GOOGLE:
        method = "Google"

        # Get the distance and duration between the two points using the Google Distance Matrix API
        distance_miles, duration_hours = google_distance_matrix_service(
            point_1=point_1,
            point_2=point_2,
            units=route_control.mileage_unit,
            organization=route_control.organization,
        )
    else:
        # If the distance method is not Google, use the Haversine formula
        method = "Monta"
        duration_hours = 0  # TODO: Implement duration calculation

        # Calculate the distance between the two points using the Haversine formula
        distance_miles = geodesic(point_1, point_2).miles

    return distance_miles, duration_hours, method


def get_shipment_mileage(*, shipment: Shipment) -> float | None:
    """
    Get the total mileage for a shipment's route or calculate the distance between two locations.

    This function attempts to retrieve a Route object from the database that matches the organization,
    origin_location, and destination_location of the shipment. If a Route object is found, the function
    returns the total_mileage attribute of the object.

    If a Route object is not found, the function calculates the distance between the shipment's origin and
    destination locations using the get_coordinates() and calculate_distance() functions. After calculating
    the distance, the function checks the organization's route_control attribute to determine if a new
    Route object should be generated using the generate_route() function. If generate_routes is True,
    the function generates a new Route object and stores it in the database.

    Args:
        shipment: An instance of the Shipment model representing a shipment to get the mileage for.

    Returns:
        A float representing the total mileage for the shipment's route if a route exists, or the calculated
        distance between the shipment's origin and destination locations.
    """

    # Get the organization's RouteControl object
    route_control = shipment.organization.route_control

    try:
        route = models.Route.objects.get(
            organization=shipment.organization,
            origin_location=shipment.origin_location,
            destination_location=shipment.destination_location,
        )
        return route.total_mileage
    except models.Route.DoesNotExist:
        # Get coordinates for two points.
        point_1, point_2 = get_coordinates(shipment=shipment)

        if not point_1 or not point_2:
            return 0

        # Calculate distance between two points.
        distance, duration, method = calculate_distance(
            point_1=point_1, point_2=point_2, route_control=route_control
        )

        # Generate a route that can be used moving forward.
        if route_control.generate_routes:
            generate_route(
                shipment=shipment, distance=distance, method=method, duration=duration
            )

        return distance
