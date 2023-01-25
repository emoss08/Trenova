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
from django.core.exceptions import ValidationError

from billing import models

pytestmark = pytest.mark.django_db


class TestDocumentClassification:
    """
    Test for Document Classification
    """

    def test_document_classification_creation(self, organization):
        """
        Test document classification creation
        """
        document_classification = models.DocumentClassification.objects.create(
            organization=organization,
            name="TEST",
            description="Test document classification",
        )

        assert document_classification.name == "TEST"
        assert document_classification.description == "Test document classification"

    def test_document_classification_update(self, document_classification):
        """
        Test document classification update
        """

        document_classification.update_doc_class(
            name="NEWDOC", description="Another Test Description"
        )

        assert document_classification.name == "NEWDOC"
        assert document_classification.description == "Another Test Description"


class TestDocumentClassificationAPI:
    """
    Test for Document Classification API
    """

    def test_get(self, api_client):
        """
        Test get Document Classification
        """
        response = api_client.get("/api/document_classifications/")
        assert response.status_code == 200

    def test_get_by_id(self, api_client, organization):
        """
        Test get Document classification by ID
        """

        _response = api_client.post(
            "/api/document_classifications/",
            {
                "organization": f"{organization}",
                "name": "test",
                "description": "Test Description",
            },
        )

        response = api_client.get(
            f"/api/document_classifications/{_response.data['id']}/"
        )
        assert response.status_code == 200
        assert response.data["name"] == "test"
        assert response.data["description"] == "Test Description"

    def test_post(self, api_client, organization):
        """
        Test Post Document Classification
        """
        response = api_client.post(
            "/api/document_classifications/",
            {
                "organization": f"{organization}",
                "name": "test",
                "description": "Test Description",
            },
            format="json",
        )

        assert response.status_code == 201
        assert response.data["name"] == "test"
        assert response.data["description"] == "Test Description"

    def test_put(self, api_client, organization):
        """
        Test Put Document Classification
        """
        _response = api_client.post(
            "/api/document_classifications/",
            {
                "organization": f"{organization}",
                "name": "test",
                "description": "Test Description",
            },
            format="json",
        )

        response = api_client.put(
            f"/api/document_classifications/{_response.data['id']}/",
            {"name": "foo", "description": "foo bar description"},
            format="json",
        )

        assert response.status_code == 200
        assert response.data["name"] == "foo"
        assert response.data["description"] == "foo bar description"

    def test_delete(self, api_client, organization):
        """
        Test Delete Document Classification
        """

        _response = api_client.post(
            "/api/document_classifications/",
            {
                "organization": f"{organization}",
                "name": "test",
                "description": "Test Description",
            },
            format="json",
        )

        response = api_client.delete(
            f"/api/document_classifications/{_response.data['id']}/"
        )

        assert response.status_code == 204
        assert response.data is None


class TestDocumentationClassificationValidation:
    """
    Test for Document Classification Validation
    """

    def test_cannot_delete_con(self, document_classification):
        """
        Test for cannot delete consolidated document classification
        """

        document_classification.name = "CON"
        document_classification.save()

        with pytest.raises(ValidationError) as excinfo:
            document_classification.delete()

        assert excinfo.value.message_dict["name"] == [
            "Document classification with this name cannot be deleted. Please try again."
        ]

    def test_unique_name(self, document_classification):
        """
        Test for unique name
        """
        with pytest.raises(ValidationError) as excinfo:
            models.DocumentClassification.objects.create(
                organization=document_classification.organization,
                name=document_classification.name,
                description="Test document classification",
            )
        assert excinfo.value.message_dict["name"] == [
            "Document classification with this name already exists. Please try again."
        ]
