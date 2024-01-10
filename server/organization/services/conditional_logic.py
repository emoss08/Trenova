# --------------------------------------------------------------------------------------------------
#  COPYRIGHT(c) 2024 MONTA                                                                         -
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
from django.apps import apps

from organization.exceptions import ConditionalStructureError
from utils.models import OperationChoices
from utils.types import ConditionalLogic

AVAILABLE_OPERATIONS = [
    "eq",
    "ne",
    "gt",
    "gte",
    "lt",
    "lte",
    "contains",
    "icontains",
    "in",
    "not_in",
    "isnull",
    "not_isnull",
]

OPERATION_MAPPING = {
    "eq": "=",
    "ne": "<>",
    "gt": ">",
    "gte": ">=",
    "lt": "<",
    "lte": "<=",
    "contains": "LIKE",
    "icontains": "ILIKE",
    "in": "IN",
    "not_in": "NOT IN",
    "isnull": "IS NULL",
    "not_isnull": "IS NOT NULL",
}


def validate_conditional_logic(*, data: ConditionalLogic) -> bool:
    required_keys = [
        "name",
        "description",
        "model_name",
        "conditions",
    ]
    for key in required_keys:
        if key not in data:
            raise ConditionalStructureError(
                f"Conditional Logic is missing required key: '{key}'"
            )

    for condition in data["conditions"]:
        for key in [
            "id",
            "column",
            "operation",
            "value",
            "data_type",
        ]:
            if key not in condition:
                raise ConditionalStructureError(
                    f"Condition is missing required key: '{key}'"
                )

        if condition["operation"] not in AVAILABLE_OPERATIONS:
            raise ConditionalStructureError(
                f"Invalid operation '{condition['operation']}' in condition ID {condition['id']}"
            )

        # Additional checks for specific operations
        if condition["operation"] in (
            OperationChoices.IN,
            OperationChoices.NOT_IN,
        ) and not isinstance(condition["value"], list):
            raise ConditionalStructureError(
                f"Operation 'in' expects a list value in condition ID {condition['id']}"
            )

        if (
            condition["operation"]
            in (OperationChoices.IS_NULL, OperationChoices.IS_NOT_NULL)
            and condition["value"] is not None
        ):
            raise ConditionalStructureError(
                f"Operation 'isnull or not_isnull' should not have a value in condition ID {condition['id']}"
            )

        if condition["operation"] in (
            OperationChoices.CONTAINS,
            OperationChoices.ICONTAINS,
        ) and not isinstance(condition["value"], str):
            raise ConditionalStructureError(
                f"Operation 'contains or icontains' expects a string value in condition ID {condition['id']}"
            )

    return True


def validate_model_fields_exist(*, data: ConditionalLogic) -> bool:
    EXCLUDED_FIELDS = [
        "id",
        "organization",
        "business_unit",
    ]

    try:
        model = apps.get_model(data["model_name"], data["app_label"])
    except LookupError as e:
        raise ConditionalStructureError(
            f"Model '{data['model_name']}' in app '{data['app_label']}' not found"
        ) from e

    fields = [field.name for field in model._meta.get_fields()]

    for conditional_field in data["conditions"]:
        conditional_field_name = conditional_field["column"]
        if conditional_field_name not in fields:
            raise ConditionalStructureError(
                f"Conditional Field '{conditional_field_name}' does not exist on model '{data['model_name']}'"
            )
        if conditional_field_name in EXCLUDED_FIELDS:
            raise ConditionalStructureError(
                f"Conditional Field '{conditional_field_name}' is not allowed for model '{data['model_name']}'"
            )

    return True
