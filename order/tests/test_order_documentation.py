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

import os

import pytest
from django.core.files.uploadedfile import SimpleUploadedFile

from billing.tests.factories import DocumentClassificationFactory
from order import models
from order.tests.factories import OrderDocumentationFactory, OrderFactory
from utils.tests import ApiTest, UnitTest


class TestOrderDocumentation(UnitTest):
    """
    Class to test Order Documentation
    """

    @pytest.fixture()
    def order_document(self):
        """
        Pytest Fixture for Order Documentation
        """
        return OrderDocumentationFactory()

    @pytest.fixture()
    def order(self):
        """
        Pytest Fixture for Order
        """
        return OrderFactory()

    @pytest.fixture()
    def document_classification(self):
        """
        Pytest Fixture for Document Classification
        """
        return DocumentClassificationFactory()

    def test_list(self, order_document):
        """
        Test Order Documentation list
        """
        assert order_document is not None

    def test_create(self, organization, order, document_classification):
        """
        Test Order Documentation Create
        """
        pdf_file = SimpleUploadedFile("dummy.pdf", b"file_content", content_type="application/pdf")

        created_document = models.OrderDocumentation.objects.create(
            organization=organization,
            order=order,
            document=pdf_file,
            document_class=document_classification,
        )

        assert created_document is not None
        assert created_document.order == order
        assert created_document.organization == organization
        assert created_document.document_class == document_classification
        assert created_document.document.name is not None
        assert created_document.document.read() == b"file_content"
        assert created_document.document.size == len(b"file_content")

    def test_update(self, order_document, organization, order, document_classification):
        """
        Test Order Documentation update
        """
        pdf_file = SimpleUploadedFile("dummy.pdf", b"file_content", content_type="application/pdf")

        updated_document = models.OrderDocumentation.objects.get(id=order_document.id)
        updated_document.document = pdf_file
        updated_document.save()

        assert updated_document is not None
        assert updated_document.document.name is not None
        assert updated_document.document.read() == b"file_content"
        assert updated_document.document.size == len(b"file_content")

class TestOrderDocumentationApi(ApiTest):
    """
    Order Documentation API
    """

    @pytest.fixture()
    def order(self):
        """
        Pytest Fixture for Order
        """
        return OrderFactory()

    @pytest.fixture()
    def document_classification(self):
        """
        Pytest Fixture for Document Classification
        """
        return DocumentClassificationFactory()

    @pytest.fixture()
    def order_documentation(
        self, api_client, order, document_classification, organization
    ):
        """
        Pytest Fixture for Order Documentation
        """

        with open("order/tests/files/dummy.pdf", "rb") as test_file:
            return api_client.post(
                "/api/order_documents/",
                {
                    "organization": f"{organization}",
                    "order": f"{order.id}",
                    "document": test_file,
                    "document_class": f"{document_classification.id}",
                },
            )

    def test_get(self, api_client):
        """
        Test get Order Documentation
        """
        response = api_client.get("/api/order_documents/")
        assert response.status_code == 200

    def test_get_by_id(
        self, api_client, order_documentation, order, document_classification
    ):
        """
        Test get Order Documentation by ID
        """

        response = api_client.get(
            f"/api/order_documents/{order_documentation.data['id']}/"
        )

        assert response.data is not None
        assert response.status_code == 200
        assert response.data["order"] == order.id
        assert response.data["document"] is not None
        assert response.data["document_class"] == document_classification.id

    def test_put(self, api_client, order, order_documentation, document_classification):
        """
        Test put Order Documentation by ID
        """

        with open("order/tests/files/dummy.pdf", "rb") as test_file:
            response = api_client.put(
                f"/api/order_documents/{order_documentation.data['id']}/",
                {
                    "order": f"{order.id}",
                    "document": test_file,
                    "document_class": f"{document_classification.id}",
                },
            )

        assert response.data is not None
        assert response.status_code == 200
        assert response.data["order"] == order.id
        assert response.data["document"] is not None
        assert response.data["document_class"] == document_classification.id

    def test_patch(
        self, api_client, order, order_documentation, document_classification
    ):
        """
        Test patch Order Documentation by ID
        """

        with open("order/tests/files/dummy.pdf", "rb") as test_file:
            response = api_client.put(
                f"/api/order_documents/{order_documentation.data['id']}/",
                {
                    "order": f"{order.id}",
                    "document": test_file,
                    "document_class": f"{document_classification.id}",
                },
            )

        assert response.data is not None
        assert response.status_code == 200
        assert response.data["order"] == order.id
        assert response.data["document"] is not None
        assert response.data["document_class"] == document_classification.id

    def test_delete(self, api_client, order_documentation):
        """
        Test Delete by ID
        """

        response = api_client.delete(
            f"/api/order_documents/{order_documentation.data['id']}/"
        )

        assert response.status_code == 204
        assert response.data is None

        if os.path.exists("testfile.txt"):
            return os.remove("testfile.txt")

    def test_tear_down(self):
        """
        Tear down tests
        """
        self.remove_media_directory("order_documentation")
