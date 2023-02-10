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

import pytest
from django.core.files.uploadedfile import SimpleUploadedFile

from order import models

pytestmark = pytest.mark.django_db


def remove_media_directory(file_path: str) -> None:
    """Remove Media Directory after test tear down.

    Primary usage is when tests are performing file uploads.
    This method deletes the media directory after the test.
    This is to prevent the media directory from filling up
    with test files.

    Args:
        file_path (str): path to directory in media folder.

    Returns:
        None
    """

    base_dir = Path(__file__).resolve().parent.parent.parent
    media_dir = os.path.join(base_dir, f"media/{file_path}")

    if os.path.exists(media_dir):
        shutil.rmtree(media_dir, ignore_errors=True, onerror=None)


def remove_file(file_path: str) -> None:
    """Remove File after test tear down.

    Primary usage is when tests are performing file uploads.
    This method deletes the file after the test.
    This is to prevent the media directory from filling up
    with test files.

    Args:
        file_path (str): path to file in media folder.

    Returns:
        None
    """

    base_dir = Path(__file__).resolve().parent.parent.parent
    file = os.path.join(base_dir, f"media/{file_path}")

    if os.path.exists(file):
        os.remove(file)


class TestOrderDocumentation:
    """
    Class to test Order Documentation
    """

    def test_list(self, order_document):
        """
        Test Order Documentation list
        """
        assert order_document is not None

    def test_create(self, organization, order, document_classification):
        """
        Test Order Documentation Create
        """
        pdf_file = SimpleUploadedFile(
            "dummy.pdf", b"file_content", content_type="application/pdf"
        )

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
        pdf_file = SimpleUploadedFile(
            "dummy.pdf", b"file_content", content_type="application/pdf"
        )

        updated_document = models.OrderDocumentation.objects.get(id=order_document.id)
        updated_document.document = pdf_file
        updated_document.save()

        assert updated_document is not None
        assert updated_document.document.name is not None
        assert updated_document.document.read() == b"file_content"
        assert updated_document.document.size == len(b"file_content")


class TestOrderDocumentationApi:
    """
    Order Documentation API
    """

    def test_get(self, api_client):
        """
        Test get Order Documentation
        """
        response = api_client.get("/api/order_documents/")
        assert response.status_code == 200

    def test_get_by_id(
        self, api_client, order_documentation_api, order, document_classification
    ):
        """
        Test get Order Documentation by ID
        """

        response = api_client.get(
            f"/api/order_documents/{order_documentation_api.data['id']}/"
        )

        assert response.data is not None
        assert response.status_code == 200
        assert response.data["order"] == order.id
        assert response.data["document"] is not None
        assert response.data["document_class"] == document_classification.id

    def test_put(
        self, api_client, order, order_documentation_api, document_classification
    ):
        """
        Test put Order Documentation by ID
        """

        with open("order/tests/files/dummy.pdf", "rb") as test_file:
            response = api_client.put(
                f"/api/order_documents/{order_documentation_api.data['id']}/",
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
        self, api_client, order, order_documentation_api, document_classification
    ):
        """
        Test patch Order Documentation by ID
        """

        with open("order/tests/files/dummy.pdf", "rb") as test_file:
            response = api_client.put(
                f"/api/order_documents/{order_documentation_api.data['id']}/",
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

    def test_delete(self, api_client, order_documentation_api):
        """
        Test Delete by I
        """

        response = api_client.delete(
            f"/api/order_documents/{order_documentation_api.data['id']}/"
        )

        assert response.status_code == 204
        assert response.data is None

        if os.path.exists("testfile.txt"):
            return os.remove("testfile.txt")

        remove_media_directory("order_documentation")
