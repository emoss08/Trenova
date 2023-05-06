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

from io import StringIO
from unittest.mock import patch

import pytest
from django.core.management import call_command

from organization import factories, models
from organization.services.psql_triggers import (
    check_function_exists,
    check_trigger_exists,
)
from organization.services.table_choices import TABLE_NAME_CHOICES
from organization.tasks import table_change_alerts

pytestmark = pytest.mark.django_db


def test_create_table_charge_alert(organization: models.Organization) -> None:
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
    assert table_charge.is_active is True
    assert table_charge.name == "Test"
    assert (
        table_charge.database_action
        == models.TableChangeAlert.DatabaseActionChoices.INSERT
    )
    assert table_charge.table == TABLE_NAME_CHOICES[0][0]


def test_table_change_insert_database_action_save() -> None:
    """
    Tests the creation of a table change alert with INSERT Action adds the proper function,
    trigger, and listener name.
    """
    table_change = factories.TableChangeAlertFactory(database_action="INSERT")

    assert table_change.function_name == f"notify_new_{table_change.table}"
    assert table_change.trigger_name == f"after_insert_{table_change.table}"
    assert table_change.listener_name == f"new_added_{table_change.table}"


def test_table_change_insert_adds_insert_trigger() -> None:
    """
    Tests that the insert trigger is added to the database when a table change alert is created
    with INSERT action.
    """
    table_change = factories.TableChangeAlertFactory(database_action="INSERT")

    trigger_check = check_trigger_exists(
        table_name=table_change.table, trigger_name=table_change.trigger_name
    )
    function_check = check_function_exists(function_name=table_change.function_name)

    print("table change", table_change)
    print("TRIGGER CHECK", trigger_check)
    print("FUNCTION CHECK", function_check)

    assert trigger_check is True
    assert function_check is True


def test_delete_table_change_removes_trigger() -> None:
    """
    Tests that the trigger is removed from the database when a table change alert is deleted.
    """
    table_change = factories.TableChangeAlertFactory(database_action="INSERT")

    trigger_check = check_trigger_exists(
        table_name=table_change.table, trigger_name=table_change.trigger_name
    )
    function_check = check_function_exists(function_name=table_change.function_name)
    assert trigger_check is True
    assert function_check is True

    table_change.delete()

    trigger_check_2 = check_trigger_exists(
        table_name=table_change.table, trigger_name=table_change.trigger_name
    )
    function_check_2 = check_function_exists(function_name=table_change.function_name)

    assert trigger_check_2 is False
    assert function_check_2 is False


def test_table_change_database_action_update() -> None:
    """
    Test changing the database action removes and adds the proper function, trigger, and listener
    names.
    """
    table_change = factories.TableChangeAlertFactory(database_action="INSERT")

    assert table_change.function_name == f"notify_new_{table_change.table}"
    assert table_change.trigger_name == f"after_insert_{table_change.table}"
    assert table_change.listener_name == f"new_added_{table_change.table}"

    table_change.database_action = "UPDATE"
    table_change.save()

    assert table_change.function_name == f"notify_updated_{table_change.table}"
    assert table_change.trigger_name == f"after_update_{table_change.table}"
    assert table_change.listener_name == f"updated_{table_change.table}"


def test_command() -> None:
    with patch("psycopg2.connect"), patch(
        "django.core.management.color.supports_color", return_value=False
    ):
        out = StringIO()
        call_command("psql_listener", stdout=out)
        assert "Starting PostgreSQL listener..." in out.getvalue()


@patch("organization.tasks.call_command")
def test_table_change_alerts_success(mock_call_command) -> None:
    table_change_alerts()
    mock_call_command.assert_called_once_with("psql_listener")
