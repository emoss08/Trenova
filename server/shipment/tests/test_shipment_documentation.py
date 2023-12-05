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
from django.core.files.base import ContentFile
from django.core.files.uploadedfile import SimpleUploadedFile
from rest_framework.response import Response
from rest_framework.test import APIClient

from billing.models import DocumentClassification
from organization.models import BusinessUnit, Organization
from shipment import models

pytestmark = pytest.mark.django_db


def test_list(shipment_document: models.ShipmentDocumentation) -> None:
    """
    Test shipment Documentation list
    """
    assert shipment_document is not None


def test_create(
    organization: Organization,
    business_unit: BusinessUnit,
    shipment: models.Shipment,
    document_classification: DocumentClassification,
) -> None:
    """
    Test shipment Documentation Create
    """

    # Create a file-like object in memory (could be any content)
    file_content = b"file_content"
    file_name = "dummy.pdf"
    content_file = ContentFile(file_content, name=file_name)

    # Create a SimpleUploadedFile using the in-memory file
    pdf_file = SimpleUploadedFile(
        name=content_file.name,
        content=content_file.read(),
        content_type="application/pdf",
    )

    # Create a shipment documentation
    created_document = models.ShipmentDocumentation.objects.create(
        organization=organization,
        business_unit=business_unit,
        shipment=shipment,
        document=pdf_file,
        document_class=document_classification,
    )

    assert created_document is not None
    assert created_document.shipment == shipment
    assert created_document.organization == organization
    assert created_document.document_class == document_classification
    assert created_document.document.name is not None
    assert created_document.document.read() == b"file_content"
    assert created_document.document.size == len(b"file_content")


def test_update(
    shipment_document: models.ShipmentDocumentation,
    organization: Organization,
    shipment: models.Shipment,
    document_classification: DocumentClassification,
) -> None:
    """
    Test shipment Documentation update
    """
    # Create a file-like object in memory (could be any content)
    file_content = b"file_content"
    file_name = "dummy.pdf"
    content_file = ContentFile(file_content, name=file_name)

    # Create a SimpleUploadedFile using the in-memory file
    pdf_file = SimpleUploadedFile(
        name=content_file.name,
        content=content_file.read(),
        content_type="application/pdf",
    )

    updated_document = models.ShipmentDocumentation.objects.get(id=shipment_document.id)
    updated_document.document = pdf_file
    updated_document.save()

    assert updated_document is not None
    assert updated_document.document.name is not None
    assert updated_document.document.read() == b"file_content"
    assert updated_document.document.size == len(b"file_content")


def test_get(api_client: APIClient):
    """
    Test get shipment Documentation
    """
    response = api_client.get("/api/shipment_documents/")
    assert response.status_code == 200


def test_get_by_id(
    api_client: APIClient,
    shipment_documentation_api: Response,
    shipment: models.Shipment,
    document_classification: DocumentClassification,
) -> None:
    """
    Test get shipment Documentation by ID
    """

    response = api_client.get(
        f"/api/shipment_documents/{shipment_documentation_api.data['id']}/"
    )

    assert response.data is not None
    assert response.status_code == 200
    assert response.data["shipment"] == shipment.id
    assert response.data["document"] is not None
    assert response.data["document_class"] == document_classification.id


def test_put(
    api_client: APIClient,
    shipment: models.Shipment,
    shipment_documentation_api: Response,
    document_classification: DocumentClassification,
) -> None:
    """
    Test put shipment Documentation by ID
    """

    # Create a file-like object in memory (could be any content)
    file_content = b"file_content"
    file_name = "dummy.pdf"
    content_file = ContentFile(file_content, name=file_name)

    # Create a SimpleUploadedFile using the in-memory file
    pdf_file = SimpleUploadedFile(
        name=content_file.name,
        content=content_file.read(),
        content_type="application/pdf",
    )

    response = api_client.put(
        f"/api/shipment_documents/{shipment_documentation_api.data['id']}/",
        {
            "shipment": f"{shipment.id}",
            "document": pdf_file,
            "document_class": f"{document_classification.id}",
        },
    )

    assert response.data is not None
    assert response.status_code == 200
    assert response.data["shipment"] == shipment.id
    assert response.data["document"] is not None
    assert response.data["document_class"] == document_classification.id


def test_patch(
    api_client: APIClient,
    shipment: models.Shipment,
    shipment_documentation_api: Response,
    document_classification: DocumentClassification,
) -> None:
    """
    Test patch shipment Documentation by ID
    """

    # Create a file-like object in memory (could be any content)
    file_content = b"file_content"
    file_name = "dummy.pdf"
    content_file = ContentFile(file_content, name=file_name)

    # Create a SimpleUploadedFile using the in-memory file
    pdf_file = SimpleUploadedFile(
        name=content_file.name,
        content=content_file.read(),
        content_type="application/pdf",
    )

    response = api_client.put(
        f"/api/shipment_documents/{shipment_documentation_api.data['id']}/",
        {
            "shipment": f"{shipment.id}",
            "document": pdf_file,
            "document_class": f"{document_classification.id}",
        },
    )

    assert response.data is not None
    assert response.status_code == 200
    assert response.data["shipment"] == shipment.id
    assert response.data["document"] is not None
    assert response.data["document_class"] == document_classification.id


def test_delete(api_client: APIClient, shipment_documentation_api: Response) -> None:
    """
    Test Delete by I
    """

    response = api_client.delete(
        f"/api/shipment_documents/{shipment_documentation_api.data['id']}/"
    )

    assert response.status_code == 204
    assert response.data is None
