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

from django.test import TestCase

from accounts.models import User, UserProfile
from organization.factories.organization import OrganizationFactory


class TestUser(TestCase):
    def setUp(self):
        self.organization = OrganizationFactory()
        self.user = User.objects.create_user(
            user_name="test_user",
            email="test@test.com",
            password="test_password",
            organization=self.organization,
        )
        self.profile = UserProfile.objects.create(
            user=self.user,
            organization=self.organization,
        )

    def test_user_creation(self):
        self.assertEqual(User.objects.count(), 1)

    def test_user_updated(self):
        self.user.user_name = "test_user_updated"
        self.user.save()
        self.assertEqual(self.user.user_name, "test_user_updated")

    def test_user_organization(self):
        self.assertEqual(self.user.organization, self.organization)

    def test_user_profile_creation(self):
        self.assertEqual(UserProfile.objects.count(), 1)

    def test_user_profile_updated(self):
        self.profile.first_name = "test_first_name"
        self.profile.save()
        self.assertEqual(self.profile.first_name, "test_first_name")
