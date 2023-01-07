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

from billing import models
from billing.tests.factories import DocumentClassificationFactory
from organization.factories import OrganizationFactory
from utils.tests import ApiTest

pytestmark = pytest.mark.django_db


class TestDocumentClassification:
    @pytest.fixture()
    def document_classification(self):
        """
        Document classification fixture
        """
        return DocumentClassificationFactory()

    @pytest.fixture()
    def organization(self):
        """
        Organization Fixture
        """
        return OrganizationFactory()

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
