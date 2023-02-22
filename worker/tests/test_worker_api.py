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

import pytest

pytestmark = pytest.mark.django_db


class TestWorkerApi:
    """
    Tests for Worker API.
    """

    def test_get(self, api_client):
        """
        Test get Document Classification
        """
        response = api_client.get("/api/workers/")
        assert response.status_code == 200

    def test_get_by_id(self, api_client, worker_api):
        """
        Test get Document classification by ID
        """

        _response = api_client.post(
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
            },
            format="json",
        )

        response = api_client.get(f"/api/workers/{worker_api.id}/")
        assert response.status_code == 200

    def test_create(self, api_client, comment_type, user):
        """
        Test creating worker
        """

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
                "profile": {
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
                        "comment": "TEST COMMENT CREATION",
                        "comment_type": comment_type.id,
                        "entered_by": user.id,
                    }
                ],
                "contacts": [
                    {
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

    def test_create_with_multi(self, api_client, comment_type, user):
        """
        Test creating worker with multiple inputs on comments,
        and contacts
        """

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
                "profile": {
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
                        "comment": "TEST COMMENT CREATION",
                        "comment_type": comment_type.id,
                        "entered_by": user.id,
                    },
                    {
                        "comment": "TEST COMMENT CREATION 2",
                        "comment_type": comment_type.id,
                        "entered_by": user.id,
                    },
                ],
                "contacts": [
                    {
                        "name": "Test Contact",
                        "phone": "1234567890",
                        "email": "test@test.com",
                        "relationship": "Mother",
                        "is_primary": True,
                        "mobile_phone": "1234567890",
                    },
                    {
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

    def test_put(self, api_client, comment_type, user, worker_api):
        """
        Test creating worker
        """

        _response = api_client.post(
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
                "profile": {
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
                        "comment": "TEST COMMENT CREATION",
                        "comment_type": comment_type.id,
                        "entered_by": user.id,
                    },
                    {
                        "comment": "TEST COMMENT CREATION 2",
                        "comment_type": comment_type.id,
                        "entered_by": user.id,
                    },
                ],
                "contacts": [
                    {
                        "name": "Test Contact",
                        "phone": "1234567890",
                        "email": "test@test.com",
                        "relationship": "Mother",
                        "is_primary": True,
                        "mobile_phone": "1234567890",
                    },
                    {
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

        response = api_client.put(
            f"/api/workers/{_response.data['id']}/",
            {
                "is_active": True,
                "worker_type": "EMPLOYEE",
                "first_name": "foo bar",
                "last_name": "bar",
                "address_line_1": "test address line 1",
                "city": "clark kent",
                "state": "CA",
                "zip_code": "12345",
                "profile": {
                    "race": "TEST",
                    "sex": "MALE",
                    "date_of_birth": "1970-12-10",
                    "license_number": "1234569780",
                    "license_state": "NC",
                    "endorsements": "N",
                },
                "comments": [
                    {
                        "id": f"{_response.data['comments'][0]['id']}",
                        "comment": "TEST COMMENT CREATION 2",
                        "comment_type": comment_type.id,
                        "entered_by": user.id,
                    }
                ],
                "contacts": [
                    {
                        "id": f"{_response.data['contacts'][0]['id']}",
                        "name": "Test Contact 2",
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
        assert response.status_code == 200
        assert response.data is not None
        assert response.data["first_name"] == "foo bar"
        assert response.data["profile"]["license_number"] == "1234569780"
        assert response.data["comments"][0]["comment"] == "TEST COMMENT CREATION 2"
        assert response.data["contacts"][0]["name"] == "Test Contact 2"
