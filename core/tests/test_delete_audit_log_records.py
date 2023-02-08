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

import datetime
from unittest.mock import patch

import pytest
from celery.exceptions import Retry

from core.tasks import delete_audit_log_records, get_cutoff_date


def test_get_cutoff_date():
    """
    Test the get_cutoff_date function.
    """
    cutoff_date: datetime.datetime = get_cutoff_date()
    assert isinstance(cutoff_date, datetime.datetime)


@patch("core.tasks.call_command")
def test_delete_audit_log_records(patched_call_command):
    """
    Test the delete_audit_log_records function.
    """
    cutoff_date: datetime.datetime = get_cutoff_date()
    formatted_date: str = cutoff_date.strftime("%Y-%m-%d")
    delete_audit_log_records()

    assert patched_call_command.call_count == 1
    assert patched_call_command.call_args_list[0][0][0] == "auditlogflush"
    assert patched_call_command.call_args_list[0][0][1] == "-b"
    assert patched_call_command.call_args_list[0][0][2] == formatted_date
    assert patched_call_command.call_args_list[0][0][3] == "-y"
    assert patched_call_command.call_args_list[0][1] == {}


@patch("core.tasks.call_command")
def test_delete_audit_log_records_retry(patched_call_command):
    """
    Test the delete_audit_log_records function.
    """
    patched_call_command.side_effect = Retry()
    with pytest.raises(Retry):
        delete_audit_log_records()


# Path: core\tests\test_delete_audit_log_records.py
