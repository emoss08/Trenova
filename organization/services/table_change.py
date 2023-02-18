"""
COPYRIGHT 2023 MONTA

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

from organization.models import TableChangeAlert
from organization.services.psql_triggers import (
    create_insert_trigger,
    create_update_trigger, drop_trigger_and_function,
)

ACTION_NAMES = {
    "INSERT": {
        "function": "notify_new",
        "trigger": "after_insert",
        "listener": "new_added",
    },
    "UPDATE": {
        "function": "notify_updated",
        "trigger": "after_update",
        "listener": "updated",
    },
    "BOTH": {
        "function": "notify_new_or_updated",
        "trigger": "after_insert_or_update",
        "listener": "new_or_updated",
    },
}


def set_trigger_name_requirements(*, instance: TableChangeAlert) -> None:
    action_names = ACTION_NAMES[instance.database_action]

    instance.function_name = f"{action_names['function']}_{instance.table}"
    instance.trigger_name = f"{action_names['trigger']}_{instance.table}"
    instance.listener_name = f"{action_names['listener']}_{instance.table}"


def create_trigger_based_on_db_action(*, instance: TableChangeAlert) -> None:
    if instance.database_action == TableChangeAlert.DatabaseActionChoices.INSERT:
        create_insert_trigger(
            trigger_name=instance.trigger_name,
            function_name=instance.function_name,
            listener_name=instance.listener_name,
            table_name=instance.table,
        )
    elif instance.database_action == TableChangeAlert.DatabaseActionChoices.UPDATE:
        create_update_trigger(
            trigger_name=instance.trigger_name,
            function_name=instance.function_name,
            listener_name=instance.listener_name,
            table_name=instance.table,
        )
    elif instance.database_action == TableChangeAlert.DatabaseActionChoices.BOTH:
        create_insert_trigger(
            trigger_name=instance.trigger_name,
            function_name=instance.function_name,
            listener_name=instance.listener_name,
            table_name=instance.table,
        )
        create_update_trigger(
            trigger_name=instance.trigger_name,
            function_name=instance.function_name,
            listener_name=instance.listener_name,
            table_name=instance.table,
        )

def drop_trigger_and_create(*, instance: TableChangeAlert) -> None:
    drop_trigger_and_function(
        function_name=instance.function_name,
        table_name=instance.table,
        trigger_name=instance.trigger_name,
    )
    create_trigger_based_on_db_action(instance=instance)