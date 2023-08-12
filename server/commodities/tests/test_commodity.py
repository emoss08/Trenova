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
from commodities import models
from django.core.exceptions import ValidationError
from rest_framework.response import Response
from rest_framework.test import APIClient

pytestmark = pytest.mark.django_db


def test_commodity_creation(commodity: models.Commodity) -> None:
    """
    Test commodity creation
    """
    assert commodity is not None


def test_unit_of_measure_choices(commodity: models.Commodity) -> None:
    """
    Test Unit of measure choices throws ValidationError
    when the passed choice is not valid.
    """
    with pytest.raises(ValidationError) as excinfo:
        commodity.unit_of_measure = "invalid"
        commodity.full_clean()

    assert excinfo.value.message_dict["unit_of_measure"] == [
        "Value 'invalid' is not a valid choice."
    ]


def test_commodity_update(commodity: models.Commodity) -> None:
    """
    Test commodity update
    """
    commodity.name = "New name"
    commodity.save()
    assert commodity.name == "New name"


def test_commodity_is_hazmat_if_hazmat_class(commodity: models.Commodity) -> None:
    """
    Test commodity hazardous material creation
    """
    assert commodity.is_hazmat == "Y"


def test_get_commodities(api_client: APIClient) -> None:
    """
    Retrieve a list of commodities

    Args:
        api_client (APIClient): API Client

    Returns:
        None: This function does not return anything.
    """

    response = api_client.get("/api/commodities/")
    assert response.status_code == 200


def test_get_commodity_by_id(api_client: APIClient, commodity_api: Response) -> None:
    """
    Retrieve a commodity by id

    Args:
        api_client (APIClient): API Client
        commodity_api (): Response from commodity_api fixture

    Returns:
        None: This function does not return anything.
    """

    response = api_client.get(f"/api/commodities/{commodity_api.data['id']}/")

    assert response.status_code == 200
    assert response.data["name"] == "TEST3"
    assert response.data["description"] == "Test Description"


def test_put_commodity(api_client: APIClient, commodity_api: Response) -> None:
    """
    Update a commodity by id

    Args:
        api_client (APIClient): API Client
        commodity_api (): Response from commodity_api fixture

    Returns:
        None: This function does not return anything.
    """

    response = api_client.put(
        f"/api/commodities/{commodity_api.data['id']}/",
        {
            "name": "TEST4",
            "description": "Test Description",
        },
    )

    assert response.status_code == 200
    assert response.data["name"] == "TEST4"
    assert response.data["description"] == "Test Description"


def test_patch_commodity(api_client: APIClient, commodity_api: Response) -> None:
    """
    Update a commodity by id

    Args:
        api_client (APIClient): API Client
        commodity_api (Response): Response from commodity_api fixture

    Returns:
        None: This function does not return anything.
    """

    response = api_client.patch(
        f"/api/commodities/{commodity_api.data['id']}/",
        {
            "name": "TEST4",
            "description": "Test Description",
        },
    )

    assert response.status_code == 200
    assert response.data["name"] == "TEST4"
    assert response.data["description"] == "Test Description"


def test_post_commodity(api_client: APIClient) -> None:
    """
    Create a commodity

    Args:
        api_client (APIClient): API Client

    Returns:
        None: This function does not return anything.
    """
    response = api_client.post(
        "/api/commodities/",
        {
            "name": "TEST3",
            "description": "Test Description",
        },
    )
    assert response.status_code == 201
    assert response.data["name"] == "TEST3"
    assert response.data["description"] == "Test Description"


def test_delete_commodity(api_client: APIClient, commodity_api: Response) -> None:
    """
    Delete a commodity by id

    Args:
        api_client (APIClient): API Client
        commodity_api (Response): Response from commodity_api fixture

    Returns:
        None: This function does not return anything.
    """

    response = api_client.delete(
        f"/api/commodities/{commodity_api.data['id']}/",
    )

    assert response.status_code == 204
