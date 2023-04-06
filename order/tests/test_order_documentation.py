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

import os
import shutil
from pathlib import Path

import pytest
from django.core.files.uploadedfile import SimpleUploadedFile
from rest_framework.response import Response
from rest_framework.test import APIClient

from billing.models import DocumentClassification
from order import models
from order.models import OrderDocumentation
from organization.models import Organization

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


def test_list(order_document: models.OrderDocumentation) -> None:
    """
    Test Order Documentation list
    """
    assert order_document is not None


def test_create(
    organization: Organization,
    order: models.Order,
    document_classification: DocumentClassification,
) -> None:
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


def test_update(
    order_document: models.OrderDocumentation,
    organization: Organization,
    order: models.Order,
    document_classification: DocumentClassification,
) -> None:
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


def test_get(api_client: APIClient):
    """
    Test get Order Documentation
    """
    response = api_client.get("/api/order_documents/")
    assert response.status_code == 200


def test_get_by_id(
    api_client: APIClient,
    order_documentation_api: Response,
    order: models.Order,
    document_classification: DocumentClassification,
) -> None:
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
    api_client: APIClient,
    order: models.Order,
    order_documentation_api: Response,
    document_classification: DocumentClassification,
) -> None:
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
    api_client: APIClient,
    order: models.Order,
    order_documentation_api: Response,
    document_classification: DocumentClassification,
) -> None:
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


def test_delete(api_client: APIClient, order_documentation_api: Response) -> None:
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
