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
from django.core.exceptions import ValidationError

from accounts import models
from accounts.tests.factories import JobTitleFactory, UserFactory
from organization.factories import OrganizationFactory

pytestmark = pytest.mark.django_db


class TestUser:
    @pytest.fixture()
    def user(self):
        """
        User fixture
        """
        return UserFactory()

    @pytest.fixture()
    def job_title(self):
        """
        Job title fixture
        """
        return JobTitleFactory()

    @pytest.fixture()
    def organization(self):
        """
        Organization fixture
        """
        return OrganizationFactory()

    def test_user_creation(self, organization):
        """
        Test user creation
        """

        user = models.User.objects.create(
            organization=organization,
            username="testuser",
            password="anothertestaccount123@",
            email="testuser@test.com",
        )
        assert user.username == "testuser"
        assert user.email == "testuser@test.com"
        assert user is not None

    def test_user_profile_update(self, user):
        """
        Test user profile update
        """

        user.profile.update_profile(  # I FORGOT I MADE THIS MAGIC METHOD :)
            first_name="foo",
            last_name="bar",
            address_line_1="foo bar line 1",
            city="foo",
            state="CA",
            zip_code="12345",
        )

        assert user.profile.first_name == "foo"
        assert user.profile.last_name == "bar"
        assert user.profile.address_line_1 == "foo bar line 1"
        assert user.profile.city == "foo"
        assert user.profile.state == "CA"
        assert user.profile.zip_code == "12345"

    def test_job_title_not_active(self, user, job_title):
        """
        Test if the job title is not active,
        that validation error is raised
        """
        job_title.is_active = False
        job_title.save()
        user.profile.title = job_title
        with pytest.raises(ValidationError, match="Title is not active"):
            user.profile.full_clean()

    def test_create_superuser(self, organization):
        """
        Test creating superuser
        """

        admin_user = models.User(
            organization=organization,
            username="test_admin",
            email="test_admin@admin.com",
            is_staff=True,
            is_superuser=True,
        )
        admin_user.set_password("test_admin")
        admin_user.save()

        assert admin_user.is_staff is True
        assert admin_user.is_superuser is True
