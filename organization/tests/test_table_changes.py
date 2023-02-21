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
from unittest.mock import patch

import pytest
from celery.exceptions import Retry
from django.core.management import call_command
from kombu.exceptions import OperationalError

from organization import factories, models
from organization.services.psql_triggers import check_trigger_exists
from organization.services.table_choices import TABLE_NAME_CHOICES
from organization.tasks import table_change_alerts

pytestmark = pytest.mark.django_db


def test_create_table_charge_alert(organization):
    """
    Tests the creation a table charge alert.
    """
    table_charge = models.TableChangeAlert.objects.create(
        organization=organization,
        is_active=True,
        name="Test",
        database_action="INSERT",
        table=TABLE_NAME_CHOICES[0][0],
    )

    assert table_charge.organization == organization
    assert table_charge.is_active == True
    assert table_charge.name == "Test"
    assert (
        table_charge.database_action
        == models.TableChangeAlert.DatabaseActionChoices.INSERT
    )
    assert table_charge.table == TABLE_NAME_CHOICES[0][0]


def test_table_change_insert_database_action_save():
    """
    Tests the creation of a table change alert with INSERT Action adds the proper function,
    trigger, and listener name.
    """
    table_change = factories.TableChangeAlertFactory(database_action="INSERT")

    assert table_change.function_name == f"notify_new_{table_change.table}"
    assert table_change.trigger_name == f"after_insert_{table_change.table}"
    assert table_change.listener_name == f"new_added_{table_change.table}"


def test_table_change_insert_adds_insert_trigger():
    """
    Tests that the insert trigger is added to the database when a table change alert is created
    with INSERT action.
    """
    table_change = factories.TableChangeAlertFactory(database_action="INSERT")

    check = check_trigger_exists(
        table_name=table_change.table, trigger_name=table_change.trigger_name
    )

    assert check == True


def test_delete_table_change_removes_trigger():
    """
    Tests that the trigger is removed from the database when a table change alert is deleted.
    """
    table_change = factories.TableChangeAlertFactory(database_action="INSERT")

    check = check_trigger_exists(
        table_name=table_change.table, trigger_name=table_change.trigger_name
    )
    assert check == True

    table_change.delete()

    check_2 = check_trigger_exists(
        table_name=table_change.table, trigger_name=table_change.trigger_name
    )
    assert check_2 == False


def test_command():
    with patch("psycopg2.connect"), patch(
        "django.core.management.color.supports_color", return_value=False
    ):
        out = StringIO()
        call_command("psql_listener", stdout=out)
        assert "Starting PostgreSQL listener..." in out.getvalue()


@patch("organization.tasks.call_command")
def test_table_change_alerts_success(mock_call_command):
    table_change_alerts()
    mock_call_command.assert_called_once_with("psql_listener")


@patch("organization.tasks.call_command")
@patch("organization.tasks.table_change_alerts.retry")
def test_table_change_alerts_failure(mock_call_command, mock_retry):
    mock_call_command.side_effect = Retry()
    mock_retry.side_effect = OperationalError()

    with pytest.raises(Retry):
        table_change_alerts()
