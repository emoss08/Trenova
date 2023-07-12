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
from rest_framework.test import APIClient

from dispatch.factories import FleetCodeFactory
from dispatch.models import CommentType
from organization.models import Organization
from worker import models
from accounts.models import User

pytestmark = pytest.mark.django_db


def test_get_worker(api_client: APIClient) -> None:
    """Test get worker

    Args:
        api_client (APIClient): Api Client

    Returns:
        None: This function does return anything.
    """
    response = api_client.get("/api/workers/")
    assert response.status_code == 200


def test_get_worker_by_id(api_client: APIClient, worker: models.Worker) -> None:
    """Test get worker by ID

    Args:
        api_client (APIClient): Api Client
        worker (Worker): Worker object

    Returns:
        None: This function does return anything.
    """

    response = api_client.get(f"/api/workers/{worker.id}/")

    assert response.status_code == 200


def test_post_worker(
    api_client: APIClient,
    comment_type: CommentType,
    user: User,
    organization: Organization,
) -> None:
    """
    Test creating worker
    """
    fleet = FleetCodeFactory()

    response = api_client.post(
        "/api/workers/",
        {
            "is_active": True,
            "worker_type": "EMPLOYEE",
            "first_name": "foo",
            "last_name": "bar",
            "address_line_1": "test address line 1",
            "city": "clark kent",
            "state": "CA",
            "zip_code": "12345",
            "manager": user.id,
            "entered_by": user.id,
            "fleet": fleet.code,
            "profile": {
                "organization": organization.id,
                "business_unit": organization.business_unit_id,
                "race": "TEST",
                "sex": "MALE",
                "date_of_birth": "1970-12-10",
                "license_number": "1234567890",
                "license_expiration_date": "2022-01-01",
                "license_state": "NC",
                "endorsements": "N",
            },
            "comments": [
                {
                    "organization": organization.id,
                    "business_unit": organization.business_unit_id,
                    "comment": "TEST COMMENT CREATION",
                    "comment_type": comment_type.id,
                    "entered_by": user.id,
                }
            ],
            "contacts": [
                {
                    "organization": organization.id,
                    "business_unit": organization.business_unit_id,
                    "name": "Test Contact",
                    "phone": "1234567890",
                    "email": "test@test.com",
                    "relationship": "Mother",
                    "is_primary": True,
                    "mobile_phone": "1234567890",
                }
            ],
        },
        format="json",
    )

    assert response.status_code == 201
    assert response.data is not None
    assert response.data["is_active"] is True
    assert response.data["worker_type"] == "EMPLOYEE"
    assert response.data["first_name"] == "foo"
    assert response.data["last_name"] == "bar"
    assert response.data["address_line_1"] == "test address line 1"
    assert response.data["city"] == "clark kent"
    assert response.data["state"] == "CA"
    assert response.data["zip_code"] == "12345"
    assert response.data["profile"]["race"] == "TEST"
    assert response.data["profile"]["sex"] == "MALE"
    assert response.data["profile"]["date_of_birth"] == "1970-12-10"
    assert response.data["profile"]["license_number"] == "1234567890"
    assert response.data["profile"]["license_state"] == "NC"
    assert response.data["profile"]["endorsements"] == "N"
    assert response.data["comments"][0]["comment"] == "TEST COMMENT CREATION"
    assert response.data["contacts"][0]["name"] == "Test Contact"
    assert response.data["contacts"][0]["phone"] == 1234567890
    assert response.data["contacts"][0]["email"] == "test@test.com"
    assert response.data["contacts"][0]["relationship"] == "Mother"
    assert response.data["contacts"][0]["is_primary"] is True
    assert response.data["contacts"][0]["mobile_phone"] == 1234567890


def test_create_with_multi(
    api_client: APIClient,
    comment_type: CommentType,
    user: User,
    organization: Organization,
) -> None:
    """
    Test creating worker with multiple inputs on comments,
    and contacts
    """
    fleet = FleetCodeFactory()

    response = api_client.post(
        "/api/workers/",
        {
            "organization": organization.id,
            "is_active": True,
            "worker_type": "EMPLOYEE",
            "first_name": "foo",
            "last_name": "bar",
            "address_line_1": "test address line 1",
            "city": "clark kent",
            "state": "CA",
            "zip_code": "12345",
            "manager": user.id,
            "entered_by": user.id,
            "fleet": fleet.code,
            "profile": {
                "organization": organization.id,
                "race": "TEST",
                "sex": "MALE",
                "date_of_birth": "1970-12-10",
                "license_number": "1234567890",
                "license_expiration_date": "2022-01-01",
                "license_state": "NC",
                "endorsements": "N",
            },
            "comments": [
                {
                    "organization": organization.id,
                    "comment": "TEST COMMENT CREATION",
                    "comment_type": comment_type.id,
                    "entered_by": user.id,
                },
                {
                    "organization": organization.id,
                    "comment": "TEST COMMENT CREATION 2",
                    "comment_type": comment_type.id,
                    "entered_by": user.id,
                },
            ],
            "contacts": [
                {
                    "organization": organization.id,
                    "name": "Test Contact",
                    "phone": "1234567890",
                    "email": "test@test.com",
                    "relationship": "Mother",
                    "is_primary": True,
                    "mobile_phone": "1234567890",
                },
                {
                    "organization": organization.id,
                    "name": "Test Contact 2",
                    "phone": "1234567890",
                    "email": "test@test.com",
                    "relationship": "Mother",
                    "is_primary": True,
                    "mobile_phone": "1234567890",
                },
            ],
        },
        format="json",
    )
    assert response.status_code == 201
    assert response.data is not None
    assert response.data["is_active"] is True
    assert response.data["worker_type"] == "EMPLOYEE"
    assert response.data["first_name"] == "foo"
    assert response.data["last_name"] == "bar"
    assert response.data["address_line_1"] == "test address line 1"
    assert response.data["city"] == "clark kent"
    assert response.data["state"] == "CA"
    assert response.data["zip_code"] == "12345"
    assert response.data["profile"]["race"] == "TEST"
    assert response.data["profile"]["sex"] == "MALE"
    assert response.data["profile"]["date_of_birth"] == "1970-12-10"
    assert response.data["profile"]["license_number"] == "1234567890"
    assert response.data["profile"]["license_state"] == "NC"
    assert response.data["profile"]["endorsements"] == "N"
    assert response.data["comments"][0]["comment"] == "TEST COMMENT CREATION"
    assert response.data["contacts"][0]["name"] == "Test Contact"
    assert response.data["contacts"][0]["phone"] == 1234567890
    assert response.data["contacts"][0]["email"] == "test@test.com"
    assert response.data["contacts"][0]["relationship"] == "Mother"
    assert response.data["contacts"][0]["is_primary"] is True
    assert response.data["contacts"][0]["mobile_phone"] == 1234567890

    # -- TEST SECOND INPUTS --
    assert response.data["comments"][1]["comment"] == "TEST COMMENT CREATION 2"
    assert response.data["contacts"][1]["name"] == "Test Contact 2"
    assert response.data["contacts"][1]["phone"] == 1234567890
    assert response.data["contacts"][1]["email"] == "test@test.com"
    assert response.data["contacts"][1]["relationship"] == "Mother"
    assert response.data["contacts"][1]["is_primary"] is True
    assert response.data["contacts"][1]["mobile_phone"] == 1234567890


def test_put(
    api_client: APIClient,
    comment_type: CommentType,
    user: User,
    worker: models.Worker,
    organization: Organization,
) -> None:
    """
    Test creating worker
    """
    fleet = FleetCodeFactory()

    _response = api_client.post(
        "/api/workers/",
        {
            "organization": organization.id,
            "is_active": True,
            "worker_type": "EMPLOYEE",
            "first_name": "foo",
            "last_name": "bar",
            "address_line_1": "test address line 1",
            "city": "clark kent",
            "state": "CA",
            "zip_code": "12345",
            "manager": user.id,
            "entered_by": user.id,
            "fleet": fleet.code,
            "profile": {
                "organization": organization.id,
                "race": "TEST",
                "sex": "MALE",
                "date_of_birth": "1970-12-10",
                "license_number": "1234567890",
                "license_expiration_date": "2022-01-01",
                "license_state": "NC",
                "endorsements": "N",
            },
            "comments": [
                {
                    "organization": organization.id,
                    "comment": "TEST COMMENT CREATION",
                    "comment_type": comment_type.id,
                    "entered_by": user.id,
                },
                {
                    "organization": organization.id,
                    "comment": "TEST COMMENT CREATION 2",
                    "comment_type": comment_type.id,
                    "entered_by": user.id,
                },
            ],
            "contacts": [
                {
                    "organization": organization.id,
                    "name": "Test Contact",
                    "phone": "1234567890",
                    "email": "test@test.com",
                    "relationship": "Mother",
                    "is_primary": True,
                    "mobile_phone": "1234567890",
                },
                {
                    "organization": organization.id,
                    "name": "Test Contact 2",
                    "phone": "1234567890",
                    "email": "test@test.com",
                    "relationship": "Mother",
                    "is_primary": True,
                    "mobile_phone": "1234567890",
                },
            ],
        },
        format="json",
    )

    payload = {
        "organization": organization.id,
        "is_active": True,
        "worker_type": "EMPLOYEE",
        "first_name": "foo bar",
        "last_name": "bar",
        "address_line_1": "test address line 1",
        "city": "clark kent",
        "state": "CA",
        "zip_code": "12345",
        "fleet": fleet.code,
        "manager": user.id,
        "entered_by": user.id,
        "profile": {
            "organization": organization.id,
            "race": "TEST",
            "sex": "MALE",
            "date_of_birth": "1970-12-10",
            "license_number": "1234569780",
            "license_state": "NC",
            "endorsements": "N",
        },
        "comments": [
            {
                "organization": organization.id,
                "id": f"{_response.data['comments'][0]['id']}",
                "comment": "TEST COMMENT CREATION 2",
                "comment_type": comment_type.id,
                "entered_by": user.id,
            }
        ],
        "contacts": [
            {
                "organization": organization.id,
                "id": f"{_response.data['contacts'][0]['id']}",
                "name": "Test Contact 2",
                "phone": "1234567890",
                "email": "test@test.com",
                "relationship": "Mother",
                "is_primary": True,
                "mobile_phone": "1234567890",
            }
        ],
    }

    response = api_client.put(
        f"/api/workers/{_response.data['id']}/",
        format="json",
        data=payload,
    )

    assert response.status_code == 200
    assert response.data is not None
    assert response.data["first_name"] == "foo bar"
    assert response.data["profile"]["license_number"] == "1234569780"
    assert response.data["comments"][0]["comment"] == "TEST COMMENT CREATION 2"
    assert response.data["contacts"][0]["name"] == "Test Contact 2"
