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

from accounts import models
from organization.factories import OrganizationFactory

pytestmark = pytest.mark.django_db


class TestUserValidation:
    @pytest.fixture()
    def organization(self):
        """
        Organization fixture
        """
        return OrganizationFactory()

    def test_create_superuser_is_superuser_error(self, organization):
        """
        Test creating superuser throws
        value error
        """

        with pytest.raises(ValueError, match="Superuser must have is_superuser=True."):
            models.User.objects.create_superuser(
                organization=organization,
                username="test_admin",
                email="test@admin.com",
                password="test_admin",
                is_superuser=False,
                is_staff=True,
            )

    def test_create_superuser_is_staff_error(self, organization):
        """
        Test creating superuser throws
        value error
        """

        with pytest.raises(ValueError, match="Superuser must have is_staff=True."):
            models.User.objects.create_superuser(
                organization=organization,
                username="test_admin",
                email="test@admin.com",
                password="test_admin",
                is_superuser=True,
                is_staff=False,
            )
