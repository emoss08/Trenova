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
from django.core.exceptions import ValidationError
from rest_framework.test import APIClient

from billing import models
from organization.models import BusinessUnit, Organization

pytestmark = pytest.mark.django_db


def test_document_classification_creation(
    organization: Organization, business_unit: BusinessUnit
) -> None:
    """
    Test document classification creation
    """
    document_classification = models.DocumentClassification.objects.create(
        organization=organization,
        business_unit=business_unit,
        name="TEST",
        description="Test document classification",
    )

    assert document_classification.name == "TEST"
    assert document_classification.description == "Test document classification"


def test_document_classification_update(
    document_classification: models.DocumentClassification,
) -> None:
    """
    Test document classification update
    """

    document_classification.update_doc_class(
        name="NEWDOC", description="Another Test Description"
    )

    assert document_classification.name == "NEWDOC"
    assert document_classification.description == "Another Test Description"


def test_get(api_client: APIClient) -> None:
    """
    Test get Document Classification
    """
    response = api_client.get("/api/document_classifications/")
    assert response.status_code == 200


def test_get_by_id(api_client: APIClient, organization: Organization) -> None:
    """
    Test get Document classification by ID
    """

    _response = api_client.post(
        "/api/document_classifications/",
        {
            "organization": organization.id,
            "name": "test",
            "description": "Test Description",
        },
    )

    response = api_client.get(f"/api/document_classifications/{_response.data['id']}/")
    assert response.status_code == 200
    assert response.data["name"] == "test"
    assert response.data["description"] == "Test Description"


def test_post(api_client: APIClient, organization: Organization) -> None:
    """
    Test Post Document Classification
    """
    response = api_client.post(
        "/api/document_classifications/",
        {
            "organization": organization.id,
            "name": "test",
            "description": "Test Description",
        },
        format="json",
    )

    assert response.status_code == 201
    assert response.data["name"] == "test"
    assert response.data["description"] == "Test Description"


def test_put(api_client: APIClient, organization: Organization) -> None:
    """
    Test Put Document Classification
    """
    _response = api_client.post(
        "/api/document_classifications/",
        {
            "organization": organization.id,
            "name": "test",
            "description": "Test Description",
        },
        format="json",
    )

    response = api_client.put(
        f"/api/document_classifications/{_response.data['id']}/",
        {
            "organization": organization.id,
            "name": "foo",
            "description": "foo bar description",
        },
        format="json",
    )

    assert response.status_code == 200
    assert response.data["name"] == "foo"
    assert response.data["description"] == "foo bar description"


def test_delete(api_client: APIClient, organization) -> None:
    """
    Test Delete Document Classification
    """

    _response = api_client.post(
        "/api/document_classifications/",
        {
            "organization": organization.id,
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


def test_cannot_delete_rate_con_doc_class(
    document_classification: models.DocumentClassification,
) -> None:
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


def test_unique_name(
    document_classification: models.DocumentClassification, business_unit: BusinessUnit
) -> None:
    """
    Test for unique name
    """
    with pytest.raises(ValidationError) as excinfo:
        models.DocumentClassification.objects.create(
            organization=document_classification.organization,
            business_unit=business_unit,
            name=document_classification.name,
            description="Test document classification",
        )
    assert excinfo.value.message_dict["name"] == [
        "Document classification with this name already exists. Please try again."
    ]
