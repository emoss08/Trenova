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

from typing import Any, Tuple

from geopy.distance import geodesic

from integration.services import google_distance_matrix_service
from order.models import Order
from organization.models import Organization
from route import models
from route.models import RouteControl


def generate_route(
    *, order: Order, distance: float, method: str, duration: str | None
) -> None:
    """
    Generate a new Route object for an order.

    This function takes an instance of the Order model representing an order, as well as the distance,
    method, and duration of the new Route object, and creates a new Route object in the database using
    the organization, origin_location, destination_location, total_mileage, duration, and distance_method attributes.

    Args:
        order: An instance of the Order model representing an order to generate a new Route object for.
        distance: A float representing the total distance of the new Route object in miles.
        method: A string representing the method used to calculate the distance ('Google' or 'Monta').
        duration: A string representing the duration of the new Route object in hours, or None if the method is 'Monta'.

    Returns:
        None: This function does not return anything.
    """

    # Create a new Route object in the database or update an existing Route object
    models.Route.objects.update_or_create(
        organization=order.organization,
        origin_location=order.origin_location,
        destination_location=order.destination_location,
        total_mileage=distance,
        duration=duration,
        distance_method=method,
    )


def get_coordinates(*, order: Order) -> tuple[tuple[float, float], tuple[float, float]]:
    """
    Retrieve the latitude and longitude coordinates for an order's origin and destination locations.

    This function takes an instance of the Order model representing an order and retrieves the latitude
    and longitude coordinates for the order's origin and destination locations. The function returns a
    tuple containing two tuples, each representing a pair of latitude and longitude coordinates for the two points.

    Args:
        order: An instance of the Order model representing an order to retrieve the coordinates for.

    Returns:
        A tuple containing two tuples, each representing a pair of latitude and longitude coordinates for
        the order's origin and destination locations.
    """
    point_1 = (order.origin_location.latitude, order.origin_location.longitude)
    point_2 = (
        order.destination_location.latitude,
        order.destination_location.longitude,
    )
    return point_1, point_2


def calculate_distance(
    *,
    organization: Organization,
    point_1: tuple[float, float],
    point_2: tuple[float, float],
) -> tuple[float | Any, str, float | None | Any]:
    """
    Calculate the distance and duration between two points on the Earth's surface using the Haversine formula or the Google Distance
    Matrix API.

    This function takes an instance of the Organization model representing the organization associated with the order, as well
    as two tuples representing the latitude and longitude coordinates of the two points to calculate the distance between.
    The function first retrieves the organization's route_control attribute to determine which method to use for calculating
    the distance. If the distance_method is set to GOOGLE, the function uses the Google Distance Matrix API to retrieve the
    distance and duration between the two points. Otherwise, the function calculates the distance using the geodesic() function
    from the geopy library, which implements the Haversine formula.

    Args:
        organization: An instance of the Organization model representing the organization associated with the order.
        point_1: A tuple of two floats representing the latitude and longitude of the first point.
        point_2: A tuple of two floats representing the latitude and longitude of the second point.

    Returns:
        A tuple containing three values: a float representing the distance between the two points in miles, a string
        representing the method used to calculate the distance ('Google' or 'Monta'), and a float representing the
        duration between the two points in hours if the method is 'Google', or None if the method is 'Monta'.
    """

    # Get the RouteControl object associated with the organization
    route_control: RouteControl = organization.route_control

    # Get the distance method from the Organization's RouteControl object
    if route_control.distance_method == RouteControl.DistanceMethodChoices.GOOGLE:
        method = "Google"

        # Get the distance and duration between the two points using the Google Distance Matrix API
        distance_miles, duration_hours = google_distance_matrix_service(
            point_1=point_1,
            point_2=point_2,
            units=route_control.mileage_unit,
            organization=organization,
        )
    else:
        # If the distance method is not Google, use the Haversine formula
        method = "Monta"
        duration_hours = 0  # TODO: Implement duration calculation

        # Calculate the distance between the two points using the Haversine formula
        distance_miles = geodesic(point_1, point_2).miles

    return distance_miles, duration_hours, method


def get_order_mileage(*, order: Order) -> float:
    """
    Get the total mileage for an order's route or calculate the distance between two locations.

    This function attempts to retrieve a Route object from the database that matches the organization,
    origin_location, and destination_location of the order. If a Route object is found, the function
    returns the total_mileage attribute of the object.

    If a Route object is not found, the function calculates the distance between the order's origin and
    destination locations using the get_coordinates() and calculate_distance() functions. After calculating
    the distance, the function checks the organization's route_control attribute to determine if a new
    Route object should be generated using the generate_route() function. If generate_routes is True,
    the function generates a new Route object and stores it in the database.

    Args:
        order: An instance of the Order model representing an order to get the mileage for.

    Returns:
        A float representing the total mileage for the order's route if a route exists, or the calculated
        distance between the order's origin and destination locations.
    """

    # Get the organization's RouteControl object
    route_control = order.organization.route_control

    try:
        route = models.Route.objects.get(
            organization=order.organization,
            origin_location=order.origin_location,
            destination_location=order.destination_location,
        )
        return route.total_mileage
    except models.Route.DoesNotExist:
        # Get coordinates for two points.
        point_1, point_2 = get_coordinates(order=order)

        # Calculate distance between two points.
        distance, duration, method = calculate_distance(
            point_1=point_1, point_2=point_2, organization=order.organization
        )

        # Generate a route that can be used moving forward.
        if route_control.generate_routes:
            generate_route(
                order=order, distance=distance, method=method, duration=duration
            )

        return distance
