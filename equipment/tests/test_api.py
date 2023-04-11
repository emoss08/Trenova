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
from django.urls import reverse
from rest_framework.response import Response
from rest_framework.test import APIClient

from equipment import models
from organization.models import Organization

pytestmark = pytest.mark.django_db


def test_equipment_type_detail_hook(equipment_type: models.EquipmentType) -> None:
    """
    Test equipment type detail is added from create_equipment_type_details_after_create Hook
    """
    assert equipment_type.equipment_type_details is not None


def test_create_equipment_type(
    api_client: APIClient, organization: Organization
) -> None:
    """
    Test create equipment type
    """

    url = "/api/equipment_types/"
    data = {
        "name": "test_equipment_type",
        "description": "Test Equipment Type Description",
    }

    response = api_client.post(url, data, format="json")
    assert response.status_code == 201
    assert response.data["name"] == "test_equipment_type"
    assert response.data["description"] == "Test Equipment Type Description"


def test_create_equip_type_with_detail(
    api_client: APIClient, organization: Organization
) -> None:
    """
    Test create equipment type with detail
    """

    url = "/api/equipment_types/"
    data = {
        "name": "test_equipment_type",
        "description": "Test Equipment Type Description",
        "equipment_type_details": {
            "equipment_class": "TRACTOR",
            "fixed_cost": 100,
            "variable_cost": 10,
            "height": 10,
            "length": 10,
            "width": 10,
            "weight": 10,
            "idling_fuel_usage": 10,
            "exempt_from_tolls": True,
        },
    }
    response = api_client.post(url, data, format="json")
    assert response.status_code == 201
    assert response.data["name"] == "test_equipment_type"
    assert response.data["description"] == "Test Equipment Type Description"
    assert response.data["equipment_type_details"]["equipment_class"] == "TRACTOR"
    assert response.data["equipment_type_details"]["exempt_from_tolls"] is True
    assert response.data["equipment_type_details"]["fixed_cost"] == "100.0000"
    assert response.data["equipment_type_details"]["variable_cost"] == "10.0000"
    assert response.data["equipment_type_details"]["height"] == "10.0000"
    assert response.data["equipment_type_details"]["length"] == "10.0000"
    assert response.data["equipment_type_details"]["width"] == "10.0000"
    assert response.data["equipment_type_details"]["weight"] == "10.0000"
    assert response.data["equipment_type_details"]["idling_fuel_usage"] == "10.0000"


def test_detail_signal_fire(api_client: APIClient, organization: Organization) -> None:
    """
    Test detail signal fire
    """

    url = "/api/equipment_types/"
    data = {
        "organization": organization.id,
        "name": "test_equipment_type",
        "description": "Test Equipment Type Description",
    }
    response = api_client.post(url, data, format="json")
    assert response.status_code == 201
    assert response.data["name"] == "test_equipment_type"
    assert response.data["description"] == "Test Equipment Type Description"
    assert response.data["equipment_type_details"] is not None


def test_update_equipment_type(
    api_client: APIClient, equipment_type_api: Response, organization: Organization
) -> None:
    """
    Test update equipment type
    """

    put_data = {
        "organization": organization.id,
        "name": "test_updated",
        "description": "Test Equipment Type Description Updated",
    }
    response = api_client.put(
        reverse("equipment-types-detail", kwargs={"pk": equipment_type_api.data["id"]}),
        put_data,
        format="json",
    )

    assert response.status_code == 200
    assert response.data["name"] == "test_updated"
    assert response.data["description"] == "Test Equipment Type Description Updated"


def test_update_equipment_details(
    api_client: APIClient, equipment_type_api: Response, organization: Organization
) -> None:
    """
    Test update equipment details
    """

    put_data = {
        "organization": organization.id,
        "name": "test_equipment_type",
        "description": "Test Equipment Updated",
        "equipment_type_details": {
            "organization": organization.id,
            "equipment_class": "TRAILER",
            "fixed_cost": "1.0000",
            "variable_cost": "0.0000",
            "height": "0.0000",
            "length": "0.0000",
            "width": "3.0000",
            "weight": "0.0000",
            "idling_fuel_usage": "0.0000",
            "exempt_from_tolls": True,
        },
    }
    response = api_client.put(
        reverse("equipment-types-detail", kwargs={"pk": equipment_type_api.data["id"]}),
        put_data,
        format="json",
    )

    assert response.status_code == 200
    assert response.data["name"] == "test_equipment_type"
    assert response.data["description"] == "Test Equipment Updated"
    assert response.data["equipment_type_details"]["equipment_class"] == "TRAILER"
    assert response.data["equipment_type_details"]["fixed_cost"] == "1.0000"
    assert response.data["equipment_type_details"]["variable_cost"] == "0.0000"
    assert response.data["equipment_type_details"]["height"] == "0.0000"
    assert response.data["equipment_type_details"]["length"] == "0.0000"
    assert response.data["equipment_type_details"]["width"] == "3.0000"
    assert response.data["equipment_type_details"]["weight"] == "0.0000"
    assert response.data["equipment_type_details"]["idling_fuel_usage"] == "0.0000"
