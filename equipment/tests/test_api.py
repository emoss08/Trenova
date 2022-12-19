"""
COPYRIGHT 2022 MONTA

This file is part of Monta.

Monta is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

Monta is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with Monta.  If not, see <https://www.gnu.org/licenses/>.
"""
from typing import OrderedDict, Any

import pytest
from rest_framework.test import APIClient

from accounts.factories import TokenFactory, UserFactory
from organization.factories import OrganizationFactory
from equipment.factories import EquipmentFactory


@pytest.fixture()
def user():
    """
    User Fixture
    """

    return UserFactory()


@pytest.fixture()
def token(user):
    """
    Token Fixture
    """

    return TokenFactory()


@pytest.fixture()
def organization(user):
    """
    Organization Fixture
    """

    return OrganizationFactory()


@pytest.fixture()
def equipment_type():
    """
    EquipmentType Fixture
    """

    return EquipmentFactory.EquipmentTypeFactory()


@pytest.fixture()
def api_client(user):
    client = APIClient()
    client.force_authenticate(user=user)
    return client


@pytest.mark.django_db
def test_create_equipment_type(api_client):
    """
    Test create equipment type
    """

    url = "/api/equipment_types/"
    data = {
        "id": "test_equipment_type",
        "description": "Test Equipment Type Description",
    }
    response = api_client.post(url, data, format="json")
    assert response.status_code == 201
    assert response.data["id"] == "test_equipment_type"
    assert response.data["description"] == "Test Equipment Type Description"


@pytest.mark.django_db
def test_create_equip_type_with_detail(api_client):
    """
    Test create equipment type with detail
    """

    url = "/api/equipment_types/"
    data = {
        "id": "test_equipment_type",
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
    assert response.data["id"] == "test_equipment_type"
    assert response.data["description"] == "Test Equipment Type Description"
    assert response.data["equipment_type_details"]["equipment_class"] == "TRACTOR"
    assert response.data["equipment_type_details"]["exempt_from_tolls"] is True


@pytest.mark.django_db
def test_detail_signal_fire(api_client):
    """
    Test detail signal fire
    """

    url = "/api/equipment_types/"
    data = {
        "id": "test_equipment_type",
        "description": "Test Equipment Type Description",
    }
    response = api_client.post(url, data, format="json")
    assert response.status_code == 201
    assert response.data["id"] == "test_equipment_type"
    assert response.data["description"] == "Test Equipment Type Description"
    assert response.data["equipment_type_details"] is not None


@pytest.mark.django_db
def test_update_equipment_type(api_client):
    """
    Test update equipment type
    """

    url = "/api/equipment_types/"
    post_data = {
        "id": "test_equipment_type",
        "description": "Test Equipment Type Description",
    }
    api_client.post(url, post_data, format="json")

    put_data = {
        "id": "test_updated",
        "description": "Test Equipment Type Description Updated",
    }
    response = api_client.put(
        f"/api/equipment_types/test_equipment_type/", put_data, format="json"
    )
    assert response.status_code == 200
    assert response.data["id"] == "test_updated"
    assert response.data["description"] == "Test Equipment Type Description Updated"


@pytest.mark.django_db
def test_update_equipment_details(api_client):
    """
    Test update equipment details
    """
    # --- Begin POST ---
    url = "/api/equipment_types/"
    post_data = {
        "id": "test_equipment_type",
        "description": "Test Equipment Type Description",
    }
    api_client.post(url, post_data, format="json")

    # --- Begin PUT ---
    put_data = {
        "id": "test_updated",
        "description": "Test Equipment Updated",
        "equipment_type_details": {
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
        f"/api/equipment_types/test_equipment_type/", put_data, format="json"
    )

    # --- ASSERTION ---
    assert response.status_code == 200
    assert response.data["id"] == "test_updated"
    assert response.data["description"] == "Test Equipment Updated"
    assert response.data["equipment_type_details"]["equipment_class"] == "TRAILER"
    assert response.data["equipment_type_details"]["exempt_from_tolls"] is True

