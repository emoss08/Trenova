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

from unittest.mock import patch

import pytest
from django.core.management import call_command
from django.db.utils import OperationalError
from psycopg2 import OperationalError as Psycopg2OperationalError

pytestmark = pytest.mark.django_db


@patch("core.management.commands.wait_for_db.Command.check")
class TestWaitForDB:
    def test_wait_for_db_ready(self, patched_check):
        patched_check.return_value = True
        call_command("wait_for_db")
        patched_check.assert_called_once_with(databases=["default"])

    @patch("time.sleep")
    def test_wait_for_db_delay(self, patched_sleep, patched_check):
        patched_check.side_effect = (
            [Psycopg2OperationalError] * 2 + [OperationalError] * 3 + [True]
        )
        call_command("wait_for_db")
        assert patched_check.call_count == 6
        patched_check.assert_called_with(databases=["default"])
