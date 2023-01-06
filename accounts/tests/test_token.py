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

from accounts.tests.factories import TokenFactory, UserFactory

from accounts import models

pytestmark = pytest.mark.django_db


class TestToken:

    @pytest.fixture()
    def token(self):
        """
        Token fixture
        """
        return TokenFactory()

    @pytest.fixture()
    def user(self):
        """
        User fixture
        """
        return UserFactory()

    def test_create(self, user):
        """
        Test token creation
        """
        new_token = models.Token.objects.create(
            user=user
        )

        assert new_token is not None
