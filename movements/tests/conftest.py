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
from rest_framework.test import APIClient

from accounts.tests.factories import TokenFactory, UserFactory
from equipment.tests.factories import EquipmentFactory
from movements.tests.factories import MovementFactory
from order.tests.factories import OrderFactory
from organization.factories import OrganizationFactory
from worker.factories import WorkerFactory

pytestmark = pytest.mark.django_db


@pytest.fixture
def token():
    """
    Token Fixture
    """
    yield TokenFactory()


@pytest.fixture
def organization():
    """
    Organization Fixture
    """
    yield OrganizationFactory()


@pytest.fixture
def user():
    """
    User Fixture
    """
    yield UserFactory()


@pytest.fixture
def api_client(token):
    """API client Fixture

    Returns:
        APIClient: Authenticated Api object
    """
    client = APIClient()
    client.credentials(HTTP_AUTHORIZATION=f"Token {token.key}")
    return client


@pytest.fixture
def movement():
    """
    Pytest Fixture for Movement
    """
    yield MovementFactory()


@pytest.fixture
def worker():
    """
    Pytest Fixture for Worker
    """
    yield WorkerFactory()


@pytest.fixture
def equipment():
    """
    Pytest fixture for Equipment
    """
    yield EquipmentFactory()


@pytest.fixture
def order():
    """
    Pytest fixture for Order
    """
    yield OrderFactory()
