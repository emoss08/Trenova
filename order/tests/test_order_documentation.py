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
import shutil
from pathlib import Path
from time import sleep

import pytest

from billing.tests.factories import DocumentClassificationFactory
from order.factories import OrderDocumentationFactory, OrderFactory
from utils.tests import UnitTest, ApiTest
from order import models


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
        test_file = os.path.basename("files/dummy.pdf")

        ord_doc = models.OrderDocumentation.objects.create(
            organization=organization,
            order=order,
            document=test_file,
            document_class=document_classification,
        )

        assert ord_doc is not None
        assert ord_doc.order == order
        assert ord_doc.document == test_file
        assert ord_doc.organization == organization
        assert ord_doc.document_class == document_classification

    def test_update(self, order_document, organization, order, document_classification):
        """
        Test Order Documentation update
        """
        test_file = os.path.basename("files/dummy.pdf")

        ord_doc = models.OrderDocumentation.objects.get(id=order_document.id)
        ord_doc.document = test_file

        ord_doc.save()

        assert ord_doc is not None
        assert ord_doc.document == test_file


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
        fpath = "testfile.txt"
        test_file = open(fpath, "w")
        test_file.write("Hello World")
        test_file.close()
        test_file = open(fpath, "r")

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

    def test_get_by_id(self, api_client, order_documentation, order, document_classification):
        """
        Test get Order Documentation by ID
        """

        response = api_client.get(
            f"/api/order_documents/{order_documentation.data['id']}/"
        )

        assert response.data is not None
        assert response.status_code == 200
        assert response.data['order'] == order.id
        assert response.data['document'] is not None
        assert response.data['document_class'] == document_classification.id

        if os.path.exists("testfile.txt"):
            # Remove file once it is generated
            return os.remove("testfile.txt")

    def test_put(self, api_client, order, order_documentation, document_classification):
        """
        Test put Order Documentation by ID
        """

        fpath = "putfile.txt"
        test_file = open(fpath, "w")
        test_file.write("Hello World")
        test_file.close()
        test_file = open(fpath, "r")

        response = api_client.put(
            f"/api/order_documents/{order_documentation.data['id']}/",
            {
                "order": f"{order.id}",
                "document": test_file,
                "document_class": f"{document_classification.id}"
            }
        )

        assert response.data is not None
        assert response.status_code == 200
        assert response.data['order'] == order.id
        assert response.data['document'] is not None
        assert response.data['document_class'] == document_classification.id

        if os.path.exists(fpath):
            # Remove file once it is generated
            return os.remove(fpath)

    def test_patch(self, api_client, order, order_documentation, document_classification):
        """
        Test patch Order Documentation by ID
        """

        fpath = "patchfile.txt"
        test_file = open(fpath, "w")
        test_file.write("Hello World")
        test_file.close()
        test_file = open(fpath, "r")

        response = api_client.put(
            f"/api/order_documents/{order_documentation.data['id']}/",
            {
                "order": f"{order.id}",
                "document": test_file,
                "document_class": f"{document_classification.id}"
            }
        )

        assert response.data is not None
        assert response.status_code == 200
        assert response.data['order'] == order.id
        assert response.data['document'] is not None
        assert response.data['document_class'] == document_classification.id

        if os.path.exists(fpath):
            # Remove file once it is generated
            return os.remove(fpath)

    def test_delete(self, api_client, order_documentation):
        """
        Test Delete by ID
        """

        response = api_client.delete(f"/api/order_documents/{order_documentation.data['id']}/")

        assert response.status_code == 204
        assert response.data is None

        if os.path.exists("testfile.txt"):
            return os.remove("testfile.txt")

    def test_tear_down(self):
        """
        Tear down tests
        """
        self.remove_media_directory("order_documentation")
