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


def test_hazardous_material_creation(
    hazardous_material: models.HazardousMaterial,
) -> None:
    """
    Test commodity hazardous material creation
    """
    assert hazardous_material is not None


def test_hazardous_class_choices(hazardous_material: models.HazardousMaterial) -> None:
    """
    Test Unit of measure choices throws ValidationError
    when the passed choice is not valid.
    """
    with pytest.raises(ValidationError) as excinfo:
        hazardous_material.hazard_class = "invalid"
        hazardous_material.full_clean()

    assert excinfo.value.message_dict["hazard_class"] == [
        "Value 'invalid' is not a valid choice."
    ]


def test_packing_group_choices(hazardous_material: models.HazardousMaterial) -> None:
    """
    Test Packing group choice throws ValidationError
    when the passed choice is not valid.
    """
    with pytest.raises(ValidationError) as excinfo:
        hazardous_material.packing_group = "invalid"
        hazardous_material.full_clean()

    assert excinfo.value.message_dict["packing_group"] == [
        "Value 'invalid' is not a valid choice."
    ]


def test_hazardous_material_update(
    hazardous_material: models.HazardousMaterial,
) -> None:
    """
    Test commodity hazardous material update
    """
    hazardous_material.name = "New name"
    hazardous_material.save()
    assert hazardous_material.name == "New name"


def test_get_hazardous_materials(
    api_client: APIClient,
) -> None:
    """
    Retrieve a list of hazardous materials

    Args:
        api_client (APIClient): Api client

    Returns:
        None: this function returns nothing.
    """
    response = api_client.get("/api/hazardous_materials/")
    assert response.status_code == 200


def test_get_hazardous_material_by_id(
    api_client: APIClient, hazardous_material_api: Response
) -> None:
    """
    Retrieve a hazardous material by id

    Args:
        api_client (APIClient): Api client
        hazardous_material_api (Response): hazardous material response

    Returns:
        None: this function returns nothing.
    """
    response = api_client.get(
        f"/api/hazardous_materials/{hazardous_material_api.data['id']}/"
    )
    assert response.status_code == 200
    assert response.data["name"] == "TEST3"
    assert response.data["description"] == "Test Description"
    assert response.data["hazard_class"] == "1.1"


def test_put_hazardous_material(
    api_client: APIClient, hazardous_material_api: Response
) -> None:
    """
    Update a hazardous material by id

    Args:
        api_client (APIClient): Api client
        hazardous_material_api (Response): hazardous material response

    Returns:
        None: this function returns nothing.
    """
    response = api_client.put(
        f"/api/hazardous_materials/{hazardous_material_api.data['id']}/",
        {
            "name": "TEST3",
            "description": "Test Description",
            "hazard_class": "1.1",
            "packing_group": "I",
        },
        format="json",
    )
    assert response.status_code == 200
    assert response.data["name"] == "TEST3"
    assert response.data["description"] == "Test Description"
    assert response.data["hazard_class"] == "1.1"
    assert response.data["packing_group"] == "I"


def test_post_hazardous_material(api_client: APIClient) -> None:
    """
    Create a hazardous material

    Args:
        api_client (APIClient): Api client

    Returns:
        None: this function returns nothing.
    """
    response = api_client.post(
        "/api/hazardous_materials/",
        {"name": "TEST3", "description": "Test Description", "hazard_class": "1.1"},
    )

    assert response.status_code == 201
    assert response.data["name"] == "TEST3"
    assert response.data["description"] == "Test Description"
    assert response.data["hazard_class"] == "1.1"


def test_patch_hazardous_material(
    api_client: APIClient, hazardous_material_api: Response
) -> None:
    """
    Update a hazardous material by id

    Args:
        api_client (APIClient): Api client
        hazardous_material_api (Response): hazardous material response

    Returns:
        None: this function returns nothing.
    """
    response = api_client.patch(
        f"/api/hazardous_materials/{hazardous_material_api.data['id']}/",
        {
            "name": "TEST3",
            "description": "Test Description",
            "hazard_class": "1.1",
            "packing_group": "I",
        },
        format="json",
    )
    assert response.status_code == 200
    assert response.data["name"] == "TEST3"
    assert response.data["description"] == "Test Description"
    assert response.data["hazard_class"] == "1.1"
    assert response.data["packing_group"] == "I"


def test_delete_hazardous_material(
    api_client: APIClient, hazardous_material_api: Response
) -> None:
    """
    Delete a hazardous material by id

    Args:
        api_client (APIClient): Api client
        hazardous_material_api (Response): hazardous material response

    Returns:
        None: this function returns nothing.
    """
    response = api_client.delete(
        f"/api/hazardous_materials/{hazardous_material_api.data['id']}/",
        format="json",
    )
    assert response.status_code == 204
