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

from billing.factories import DocumentClassificationFactory


@pytest.fixture()
def document_classification():
    """
    Document classification fixture
    """
    return DocumentClassificationFactory()


@pytest.mark.django_db
def test_document_classification_creation(document_classification):
    """
    Test document classification creation
    """
    assert document_classification is not None


@pytest.mark.django_db
def test_document_classification_update(document_classification):
    """
    Test document classification update
    """
    document_classification.name = "New name"
    document_classification.save()
    assert document_classification.name == "New name"
