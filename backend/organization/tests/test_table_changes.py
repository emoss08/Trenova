# --------------------------------------------------------------------------------------------------
#  COPYRIGHT(c) 2024 Trenova                                                                       -
#                                                                                                  -
#  This file is part of Trenova.                                                                   -
#                                                                                                  -
#  The Trenova software is licensed under the Business Source License 1.1. You are granted the right
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
from django.core.exceptions import ValidationError
from django.core.management import call_command

from kafka.managers import KafkaManager
from organization import factories, models
from organization.exceptions import ConditionalStructureError
from organization.models import TableChangeAlert
from organization.services.conditional_logic import (
    validate_conditional_logic,
    validate_model_fields_exist,
)
from organization.services.psql_triggers import (
    _get_routine_definition,
    build_conditional_logic_sql,
    check_function_exists,
    check_trigger_exists,
)
from organization.services.table_choices import TABLE_NAME_CHOICES

pytestmark = pytest.mark.django_db


def test_create_table_charge_alert(organization: models.Organization) -> None:
    """Tests the creation a table charge alert.

    Returns:
        None: This function does not return anything.
    """
    table_charge = models.TableChangeAlert.objects.create(
        business_unit=organization.business_unit,
        organization=organization,
        status="A",
        name="Test",
        database_action="INSERT",
        email_recipients="admin@trenova.app",
        table=TABLE_NAME_CHOICES[0][0],
    )

    assert table_charge.organization == organization
    assert table_charge.status == "A"
    assert table_charge.name == "Test"
    assert (
        table_charge.database_action
        == models.TableChangeAlert.DatabaseActionChoices.INSERT
    )
    assert table_charge.table == TABLE_NAME_CHOICES[0][0]


def test_table_change_insert_database_action_save() -> None:
    """Tests the creation of a table change alert with INSERT Action adds the proper function,
    trigger, and listener name.

    Returns:
        None: This function does not return anything.
    """
    table_change = factories.TableChangeAlertFactory(database_action="INSERT")

    assert table_change.function_name == f"notify_new_{table_change.table}"
    assert table_change.trigger_name == f"after_insert_{table_change.table}"
    assert table_change.listener_name == f"new_added_{table_change.table}"


def test_table_change_insert_adds_insert_trigger() -> None:
    """Tests that the insert trigger is added to the database when a table change alert is created
    with INSERT action.

    Returns:
        None: This function does not return anything.
    """
    table_change = factories.TableChangeAlertFactory(database_action="INSERT")

    trigger_check = check_trigger_exists(
        table_name=table_change.table, trigger_name=table_change.trigger_name
    )
    function_check = check_function_exists(function_name=table_change.function_name)

    assert trigger_check is True
    assert function_check is True


def test_delete_table_change_removes_trigger() -> None:
    """
    Tests that the trigger is removed from the database when a table change alert is deleted.

    Returns:
        None: This function does not return anything.
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


def test_conditional_log_drops_and_recreates_trigger() -> None:
    """
    Tests that the trigger is removed from the database when a table change alert is deleted.

    Returns:
        None: This function does not return anything.
    """
    table_change = factories.TableChangeAlertFactory(database_action="INSERT")

    trigger_check = check_trigger_exists(
        table_name=table_change.table, trigger_name=table_change.trigger_name
    )
    function_check = check_function_exists(function_name=table_change.function_name)
    assert trigger_check is True
    assert function_check is True

    table_change.conditional_logic = {
        "name": "Join Customer and Shipment Condition",
        "description": "Send out table change alert when Customer ID is 123",
        "model_name": "shipment",
        "app_label": "Shipment",
        "conditions": [
            {
                "id": 1,
                "model_name": "customer",
                "app_label": "Customer",
                "column": "id",
                "operation": "eq",
                "value": "123",
                "data_type": "string",
            }
        ],
    }
    table_change.save()

    trigger_check_2 = check_trigger_exists(
        table_name=table_change.table, trigger_name=table_change.trigger_name
    )
    function_check_2 = check_function_exists(function_name=table_change.function_name)

    assert trigger_check_2 is True
    assert function_check_2 is True


def test_table_change_database_action_update() -> None:
    """Test changing the database action removes and adds the proper function, trigger, and listener
    names.

    Returns:
        None: This function does not return anything.
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
    """Tests that the psql_listener command runs successfully.

    Returns:
        None: This function does not return anything.
    """
    with (
        patch("psycopg.connect"),
        patch("django.core.management.color.supports_color", return_value=False),
    ):
        out = StringIO()
        call_command("psql_listener", stdout=out)
        assert "Starting PostgreSQL listener..." in out.getvalue()


def test_save_table_change_alert_kafka_without_topic(
    organization: models.Organization,
) -> None:
    """Tests that a ValidationError is raised when trying to save a TableChangeAlert with source as
    ``Kafka`` but no topic.

    Returns:
        None: This function does not return anything.
    """
    # Create a TableChangeAlert instance with source as Kafka but no topic
    kafka_alert = TableChangeAlert(source=TableChangeAlert.SourceChoices.KAFKA)

    # Expect a ValidationError when trying to save
    with pytest.raises(ValidationError) as excinfo:
        kafka_alert.clean()

    # Check if the error message is correct
    assert excinfo.value.message_dict["topic"] == [
        "Topic is required when source is Kafka."
    ]


def test_save_table_change_alert_postgres_without_table(
    organization: models.Organization,
) -> None:
    """Tests that a ValidationError is raised when trying to save a TableChangeAlert with source as
    Postgres but no table.

    Args:
        organization (models.Organization); Organization instance.

    Returns:
        None: This function does not return anything.
    """
    # Create a TableChangeAlert instance with source as Postgres but no table
    alert = TableChangeAlert(
        organization=organization,
        source=TableChangeAlert.SourceChoices.POSTGRES,
    )

    # Expect a ValidationError when trying to save
    with pytest.raises(ValidationError) as excinfo:
        alert.clean()

    # Check if the error message is correct
    assert excinfo.value.message_dict["table"] == [
        "Table is required when source is Postgres."
    ]


def test_cannot_save_if_kafka_offline(organization: models.Organization) -> None:
    """Test validationError is thrown if source is ``KAFKA`` and Kafka is offline.

    Args:
        organization (models.Organization): Organization instance.

    Returns:
        None: This function does not return anything.

    Notes:
        This test requires Kafka to be offline. If Kafka is online, this test will fail.
    """

    manager = KafkaManager()

    if manager.is_kafka_available():
        pytest.skip("Kafka is online. Skipping test.")

    alert = TableChangeAlert(
        organization=organization,
        source=TableChangeAlert.SourceChoices.KAFKA,
        topic="test",
    )

    with pytest.raises(ValidationError) as excinfo:
        alert.clean()

    assert excinfo.value.message_dict["source"] == [
        f"Unable to connect to Kafka at {manager.kafka_host}:{manager.kafka_port}."
        " Please check your connection and try again."
    ]


def test_cannot_save_delete_if_source_not_kafka(
    organization: models.Organization,
) -> None:
    """Test ValidationError is thrown if ``database_action`` is ``delete`` and source is not ``KAFKA``.

    Args:
        organization (models.Organization): Organization instance.

    Returns:
        None: This function does not return anything.
    """

    alert = TableChangeAlert(
        organization=organization,
        source=TableChangeAlert.SourceChoices.POSTGRES,
        table=TABLE_NAME_CHOICES[0][0],
        database_action=TableChangeAlert.DatabaseActionChoices.DELETE,
    )
    with pytest.raises(ValidationError) as excinfo:
        alert.clean()

    assert excinfo.value.message_dict["database_action"] == [
        "Database action can only be delete when source is Kafka."
        " Please change the source to Kafka and try again."
    ]


def test_conditional_is_missing_conditions_key() -> None:
    """Tests that a ConditionalStructureError is raised when a required key is missing from the
    conditional logic.

    Returns:
        None: This function does not return anything.
    """

    data = {
        "name": "Join Customer and Shipment Condition",
        "description": "Send out table change alert when Customer ID is 123",
        "model_name": "shipment",
        "app_label": "Shipment",
    }

    with pytest.raises(ConditionalStructureError) as excinfo:
        validate_conditional_logic(data=data)

    assert (
        excinfo.value.args[0]
        == "Conditional Logic is missing required key: 'conditions'"
    )


def test_conditional_has_invalid_operation() -> None:
    """Tests that a ConditionalStructureError is raised when the operation is invalid.

    Returns:
        None: This function does not return anything.
    """

    data = {
        "name": "Join Customer and Shipment Condition",
        "description": "Send out table change alert when Customer ID is 123",
        "model_name": "shipment",
        "app_label": "Shipment",
        "conditions": [
            {
                "id": 1,
                "model_name": "customer",
                "app_label": "Customer",
                "column": "id",
                "operation": "equals",
                "value": "123",
                "data_type": "string",
            }
        ],
    }

    with pytest.raises(ConditionalStructureError) as excinfo:
        validate_conditional_logic(data=data)

    assert excinfo.value.args[0] == "Invalid operation 'equals' in condition ID 1"


def test_conditional_is_valid_structure() -> None:
    """Tests that a ConditionalStructureError is not raised when the conditional logic is valid.

    Returns:
        None: This function does not return anything.
    """

    data = {
        "name": "Join Customer and Shipment Condition",
        "description": "Send out table change alert when Customer ID is 123",
        "model_name": "shipment",
        "app_label": "Shipment",
        "conditions": [
            {
                "id": 1,
                "model_name": "customer",
                "app_label": "Customer",
                "column": "id",
                "operation": "eq",
                "value": "123",
                "data_type": "string",
            }
        ],
    }

    assert validate_conditional_logic(data=data) is True


def test_conditional_in_operation_valid_list() -> None:
    """Tests that a ConditionalStructureError is raised when the operation is invalid.

    Returns:
        None: This function does not return anything.
    """

    data = {
        "name": "Join Customer and Shipment Condition",
        "description": "Send out table change alert when Customer ID is 123",
        "model_name": "shipment",
        "app_label": "Shipment",
        "conditions": [
            {
                "id": 1,
                "model_name": "customer",
                "app_label": "Customer",
                "column": "id",
                "operation": "in",
                "value": ["123", "456"],
                "data_type": "string",
            }
        ],
    }

    assert validate_conditional_logic(data=data) is True


def test_conditional_in_operation_requires_list() -> None:
    """Tests that a ConditionalStructureError is raised when the operation is invalid.

    Returns:
        None: This function does not return anything.
    """

    data = {
        "name": "Join Customer and Shipment Condition",
        "description": "Send out table change alert when Customer ID is 123",
        "model_name": "shipment",
        "app_label": "Shipment",
        "conditions": [
            {
                "id": 1,
                "model_name": "customer",
                "app_label": "Customer",
                "column": "id",
                "operation": "in",
                "value": "123",
                "data_type": "string",
            }
        ],
    }

    with pytest.raises(ConditionalStructureError) as excinfo:
        validate_conditional_logic(data=data)

    assert (
        excinfo.value.args[0] == "Operation 'in' expects a list value in condition ID 1"
    )


def test_conditional_isnull_operation_requires_null() -> None:
    """Tests that a ConditionalStructureError is raised when the operation is invalid.

    Returns:
        None: This function does not return anything.
    """

    data = {
        "name": "Join Customer and Shipment Condition",
        "description": "Send out table change alert when Customer ID is 123",
        "model_name": "shipment",
        "app_label": "Shipment",
        "conditions": [
            {
                "id": 1,
                "column": "id",
                "operation": "isnull",
                "value": "123",
                "data_type": "string",
            }
        ],
    }

    with pytest.raises(ConditionalStructureError) as excinfo:
        validate_conditional_logic(data=data)

    assert (
        excinfo.value.args[0]
        == "Operation 'isnull or not_isnull' should not have a value in condition ID 1"
    )


def test_conditional_contains_operation_requires_string() -> None:
    """Tests that a ConditionalStructureError is raised when the operation is invalid.

    Returns:
        None: This function does not return anything.
    """

    data = {
        "name": "Join Customer and Shipment Condition",
        "description": "Send out table change alert when Customer ID is 123",
        "model_name": "shipment",
        "app_label": "Shipment",
        "conditions": [
            {
                "id": 1,
                "column": "id",
                "operation": "contains",
                "value": 1,
                "data_type": "string",
            }
        ],
    }

    with pytest.raises(ConditionalStructureError) as excinfo:
        validate_conditional_logic(data=data)

    assert (
        excinfo.value.args[0]
        == "Operation 'contains or icontains' expects a string value in condition ID 1"
    )


def test_validate_model_fields_exist() -> None:
    """Tests that a ConditionalStructureError is not raised when the conditional logic is valid.

    Returns:
        None: This function does not return anything.
    """

    data = {
        "name": "Join Customer and Shipment Condition",
        "description": "Send out table change alert when Customer ID is 123",
        "model_name": "shipment",
        "app_label": "shipment",
        "conditions": [
            {
                "id": 1,
                "column": "pro_number",
                "operation": "eq",
                "value": "123",
                "data_type": "string",
            }
        ],
    }

    assert validate_model_fields_exist(data=data) is True


def test_validate_model_fields_with_invalid_app_label() -> None:
    """Test ConditionalStructureError is raised when app_label is invalid.

    Returns:
        None: This function does not return anything.
    """

    data = {
        "name": "Join Customer and Shipment Condition",
        "description": "Send out table change alert when Customer ID is 123",
        "model_name": "shipment",
        "app_label": "Test",
        "conditions": [
            {
                "id": 1,
                "column": "id",
                "operation": "eq",
                "value": "123",
                "data_type": "string",
            }
        ],
    }

    with pytest.raises(ConditionalStructureError) as excinfo:
        validate_model_fields_exist(data=data)

    assert excinfo.value.args[0] == "Model 'shipment' in app 'Test' not found"


def test_validate_model_fields_with_invalid_conditional() -> None:
    """Test ConditionalStructureError is raised when app_label is invalid.

    Returns:
        None: This function does not return anything.
    """
    data = {
        "name": "Join Customer and Shipment Condition",
        "description": "Send out table change alert when Customer ID is 123",
        "model_name": "shipment",
        "app_label": "shipment",
        "conditions": [
            {
                "id": 1,
                "column": "test",
                "operation": "eq",
                "value": "123",
                "data_type": "string",
            }
        ],
    }

    with pytest.raises(ConditionalStructureError) as excinfo:
        validate_model_fields_exist(data=data)

    assert (
        excinfo.value.args[0]
        == "Conditional Field 'test' does not exist on model 'shipment'"
    )


def test_validate_model_field_with_excluded_field() -> None:
    """Test ConditionalStructureError is raised when the join field used is excluded.

    Returns:
        None: This function does not return anything.
    """
    data = {
        "name": "Join Customer and Shipment Condition",
        "description": "Send out table change alert when Customer ID is 123",
        "model_name": "shipment",
        "app_label": "shipment",
        "conditions": [
            {
                "id": 1,
                "column": "organization",
                "operation": "eq",
                "value": "123",
                "data_type": "string",
            }
        ],
    }

    with pytest.raises(ConditionalStructureError) as excinfo:
        validate_model_fields_exist(data=data)

    assert (
        excinfo.value.args[0]
        == "Conditional Field 'organization' is not allowed for model 'shipment'"
    )


def test_build_eq_conditional_logic_sql() -> None:
    """Validate that the conditional logic is built correctly for eq operation.

    Returns:
        None: This function does not return anything.
    """
    data = {
        "name": "Join Customer and Shipment Condition",
        "description": "Send out table change alert when Customer ID is 123",
        "model_name": "shipment",
        "app_label": "shipment",
        "conditions": [
            {
                "id": 1,
                "column": "organization",
                "operation": "eq",
                "value": "123",
                "data_type": "string",
            }
        ],
    }

    sql = build_conditional_logic_sql(conditional_logic=data)

    assert sql == "new.organization = '123'"


def test_build_isnull_conditional_logic_sql() -> None:
    """Validate that the conditional logic is built correctly for isnull operation.

    Returns:
        None: This function does not return anything.
    """
    data = {
        "name": "Join Customer and Shipment Condition",
        "description": "Send out table change alert when Customer ID is 123",
        "model_name": "shipment",
        "app_label": "shipment",
        "conditions": [
            {
                "id": 1,
                "column": "organization",
                "operation": "isnull",
                "value": None,
                "data_type": "string",
            }
        ],
    }

    sql = build_conditional_logic_sql(conditional_logic=data)

    assert sql == "new.organization IS NULL"


def test_build_less_than_conditional_logic_sql() -> None:
    """Validate that the conditional logic is built correctly for less than operation.

    Returns:
        None: This function does not return anything.
    """
    data = {
        "name": "Join Customer and Shipment Condition",
        "description": "Send out table change alert when Customer ID is 123",
        "model_name": "shipment",
        "app_label": "shipment",
        "conditions": [
            {
                "id": 1,
                "column": "organization",
                "operation": "lt",
                "value": "123",
                "data_type": "string",
            }
        ],
    }

    sql = build_conditional_logic_sql(conditional_logic=data)

    assert sql == "new.organization < '123'"


def test_greater_than_conditional_logic_sql() -> None:
    """Validate that the conditional logic is built correctly for greater than operation.

    Returns:
        None: This function does not return anything.
    """
    data = {
        "name": "Join Customer and Shipment Condition",
        "description": "Send out table change alert when Customer ID is 123",
        "model_name": "shipment",
        "app_label": "shipment",
        "conditions": [
            {
                "id": 1,
                "column": "organization",
                "operation": "gt",
                "value": "123",
                "data_type": "string",
            }
        ],
    }

    sql = build_conditional_logic_sql(conditional_logic=data)

    assert sql == "new.organization > '123'"


def test_build_contains_conditional_logic_sql() -> None:
    """Validate that the conditional logic is built correctly for contains operation.

    Returns:
        None: This function does not return anything.
    """
    data = {
        "name": "Join Customer and Shipment Condition",
        "description": "Send out table change alert when Customer ID contains 123",
        "model_name": "shipment",
        "app_label": "shipment",
        "conditions": [
            {
                "id": 1,
                "column": "organization",
                "operation": "contains",
                "value": "123",
                "data_type": "string",
            }
        ],
    }

    sql = build_conditional_logic_sql(conditional_logic=data)

    assert sql == "new.organization LIKE '%123%'"


def test_build_icontains_conditional_logic_sql() -> None:
    """Validate that the conditional logic is built correctly for icontains operation.

    Returns:
        None: This function does not return anything.
    """
    data = {
        "name": "Join Customer and Shipment Condition",
        "description": "Send out table change alert when Customer ID contains 123",
        "model_name": "shipment",
        "app_label": "shipment",
        "conditions": [
            {
                "id": 1,
                "column": "organization",
                "operation": "icontains",
                "value": "123",
                "data_type": "string",
            }
        ],
    }

    sql = build_conditional_logic_sql(conditional_logic=data)

    assert sql == "new.organization ILIKE '%123%'"


def test_build_in_conditional_logic_sql() -> None:
    """Validate that the conditional logic is built correctly for in operation.

    Returns:
        None: This function does not return anything.
    """
    data = {
        "name": "Join Customer and Shipment Condition",
        "description": "Send out table change alert when Customer ID contains 123",
        "model_name": "shipment",
        "app_label": "shipment",
        "conditions": [
            {
                "id": 1,
                "column": "organization",
                "operation": "in",
                "value": ["123", "456"],
                "data_type": "string",
            }
        ],
    }

    sql = build_conditional_logic_sql(conditional_logic=data)

    assert sql == "new.organization IN ('123','456')"


def test_build_not_in_conditional_logic_sql() -> None:
    """Validate that the conditional logic is built correctly for not_in operation.

    Returns:
        None: This function does not return anything.
    """
    data = {
        "name": "Join Customer and Shipment Condition",
        "description": "Send out table change alert when Customer ID contains 123",
        "model_name": "shipment",
        "app_label": "shipment",
        "conditions": [
            {
                "id": 1,
                "column": "organization",
                "operation": "not_in",
                "value": ["123", "456"],
                "data_type": "string",
            }
        ],
    }

    sql = build_conditional_logic_sql(conditional_logic=data)

    assert sql == "new.organization NOT IN ('123','456')"


def test_build_not_isnull_conditional_logic_sql() -> None:
    """Validate that the conditional logic is built correctly for not_isnull operation.

    Returns:
        None: This function does not return anything.
    """
    data = {
        "name": "Join Customer and Shipment Condition",
        "description": "Send out table change alert when organization is not null",
        "model_name": "shipment",
        "app_label": "shipment",
        "conditions": [
            {
                "id": 1,
                "column": "organization",
                "operation": "not_isnull",
                "value": None,
                "data_type": "string",
            }
        ],
    }

    sql = build_conditional_logic_sql(conditional_logic=data)

    assert sql == "new.organization IS NOT NULL"


def test_create_insert_function_with_conditional() -> None:
    """Test that the insert function is created with conditional logic.

    Returns:
        None: This function does not return anything.
    """
    table_change = factories.TableChangeAlertFactory(
        database_action="INSERT",
        conditional_logic={
            "name": "Join Customer and Shipment Condition",
            "description": "Send out table change alert when Customer ID contains 123",
            "model_name": "shipment",
            "app_label": "shipment",
            "conditions": [
                {
                    "id": 1,
                    "column": "organization",
                    "operation": "contains",
                    "value": "123",
                    "data_type": "string",
                }
            ],
        },
    )

    trigger_check = check_trigger_exists(
        table_name=table_change.table, trigger_name=table_change.trigger_name
    )
    function_check = check_function_exists(function_name=table_change.function_name)

    assert trigger_check is True
    assert function_check is True


def test_routine_definition_contains_conditional() -> None:
    """Test the routine definition for the table change alert contains the proper conditional
    logic.

    Returns:
        None: This function does not return anything.
    """
    table_change = factories.TableChangeAlertFactory(
        database_action="INSERT",
        conditional_logic={
            "name": "Join Customer and Shipment Condition",
            "description": "Send out table change alert when Customer ID contains 123",
            "model_name": "shipment",
            "app_label": "shipment",
            "conditions": [
                {
                    "id": 1,
                    "column": "organization",
                    "operation": "contains",
                    "value": "123",
                    "data_type": "string",
                }
            ],
        },
    )

    route_definition = _get_routine_definition(routine_name=table_change.function_name)

    assert (
        f"IF TG_OP = 'INSERT' AND NEW.organization_id = '{table_change.organization_id}' AND (new.organization LIKE '%123%')"
        in route_definition
    )
