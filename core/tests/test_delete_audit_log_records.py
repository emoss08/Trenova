# --------------------------------------------------------------------------------------------------
#  COPYRIGHT(c) 2023 MONTA                                                                         -
#                                                                                                  -
#  This file is part of Monta.                                                                     -
#                                                                                                  -
#  The Monta software is licensed under the Business Source License 1.1. You are granted the right -
#  to copy, modify, and redistribute the software, but only for non-production use or with a total -
#  of less than three server instances. Starting from the Change Date (November 16, 2026), the     -
#  software will be made available under version 2 or later of the GNU General Public License.     -
#  If you use the software in violation of this license, your rights under the license will be     -
#  terminated automatically. The software is provided "as is," and the Licensor disclaims all      -
#  warranties and conditions. If you use this license's text or the "Business Source License" name -
#  and trademark, you must comply with the Licensor's covenants, which include specifying the      -
#  Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use     -
#  Grant, and not modifying the license in any other way.                                          -
# --------------------------------------------------------------------------------------------------

from unittest.mock import patch

import pytest
from celery.exceptions import Retry

from core.tasks import delete_audit_log_records


@patch("core.tasks.call_command")
def test_delete_audit_log_records(patched_call_command):
    """
    Test the delete_audit_log_records function.
    """
    delete_audit_log_records()

    assert patched_call_command.call_count == 1
    assert patched_call_command.call_args_list[0][0][0] == "auditlogflush"
    assert patched_call_command.call_args_list[0][0][1] == "-b"
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


