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
from io import StringIO

import pytest
from django.core.management import call_command
from django.test import TestCase

pytestmark = pytest.mark.django_db


class TestCreateSystemUser:
    def call_command(self, *args, **kwargs) -> str:
        out = StringIO()
        call_command(
            "createsystemuser",
            *args,
            stdout=out,
            stderr=StringIO(),
            **kwargs,
        )
        return out.getvalue()

    def test_create_system_user(self) -> None:
        """
        Test create system user.

        Returns:
            None: None
        """

        out = self.call_command()
        assert out == "\x1b[32;1mSystem user account created!\x1b[0m\n"

    def test_create_system_user_with_arguments(self) -> None:
        """
        Test create system user with arguments.

        Returns:
            None: None
        """

        out = self.call_command(
            "--username", "sys",
            "--email", "system@monta.io",
            "--password", "password",
            "--organization", "Monta",
        )

        assert out == "\x1b[32;1mSystem user account created!\x1b[0m\n"
