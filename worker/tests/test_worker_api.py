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

from dispatch.factories import CommentTypeFactory
from utils.tests import ApiTest
from worker.factories import WorkerFactory

pytestmark = pytest.mark.django_db


class TestWorkerApi(ApiTest):
    """
    Tests for Worker API.
    """

    @pytest.fixture()
    def worker(self):
        """
        Worker Fixture
        """

        return WorkerFactory()

    @pytest.fixture()
    def comment_type(self):
        """
        Comment Type Fixture
        """

        return CommentTypeFactory()

    def test_get(self, api_client):
        """
        Test get Document Classification
        """
        response = api_client.get("/api/workers/")
        assert response.status_code == 200

    def test_get_by_id(self, api_client, worker):
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

        response = api_client.get(f"/api/workers/{worker.id}/")

        assert response.status_code == 200

    def test_create(self, api_client):
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
            },
            format="json",
        )

        assert response.status_code == 201
        assert response.data["is_active"] is True
        assert response.data["worker_type"] == "EMPLOYEE"
        assert response.data["first_name"] == "foo"
        assert response.data["last_name"] == "bar"
        assert response.data["address_line_1"] == "test address line 1"
        assert response.data["city"] == "clark kent"
        assert response.data["state"] == "CA"
        assert response.data["zip_code"] == "12345"

    def test_create_worker_with_profile(self, api_client):
        """
        Test creating worker with profile
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
                "profile": {
                    "race": "TEST",
                    "sex": "male",
                    "date_of_birth": "1970-12-10",
                    "license_number": "1234567890",
                    "license_state": "NC",
                    "endorsements": "N",
                },
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
        assert response.data["profile"]["sex"] == "male"
        assert response.data["profile"]["date_of_birth"] == "1970-12-10"
        assert response.data["profile"]["license_number"] == "1234567890"
        assert response.data["profile"]["license_state"] == "NC"
        assert response.data["profile"]["endorsements"] == "N"

    def test_create_worker_with_comments(self, api_client, comment_type, user):
        """
        Test creating worker with comments
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
                "comments": [
                    {
                        "comment": "TEST COMMENT CREATION",
                        "comment_type": comment_type.id,
                        "entered_by": user.id,
                    }
                ],
            },
            format="json",
        )

        assert response.status_code == 201
        assert response.data["comments"][0]["comment"] == "TEST COMMENT CREATION"
