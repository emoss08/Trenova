# Copyright (c) 2024 Trenova Technologies, LLC
#
# Licensed under the Business Source License 1.1 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://trenova.app/pricing/
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#
# Key Terms:
# - Non-production use only
# - Change Date: 2026-11-16
# - Change License: GNU General Public License v2 or later
#
# For full license text, see the LICENSE file in the root directory.

import json
import os
from typing import Dict, List, Optional
import requests
from celery import Celery
from sqlalchemy import update
from report.schemas import Relationship
from report.service import report_generate
from database import SessionLocal, TaskStatus


celery = Celery(__name__)
celery.conf.broker_url = os.environ.get("CELERY_BROKER_URL", "redis://localhost:6379/0")
celery.conf.result_backend = os.environ.get(
    "CELERY_RESULT_BACKEND", "redis://localhost:6379/0"
)


def notify_go_app(
    task_id, user_id, organization_id, business_unit_id, status, result=None, error=None
) -> None:
    """Notify the Go application about the status of a Celery task.

    This function sends a notification to the Go application with the task status,
    result, and any error information. The notification is sent as a POST request
    to the Go application's WebSocket endpoint.

    Args:
        task_id (str): The ID of the Celery task.
        user_id (str): The ID of the user associated with the task.
        status (str): The status of the task (e.g., 'completed', 'failed').
        result (str, optional): The result of the task, if applicable. Defaults to None.
        error (str, optional): The error message, if applicable. Defaults to None.

    Raises:
        requests.RequestException: If an error occurs while sending the notification
        to the Go application.

    Side Effects:
        - Sends a POST request to the Go application's WebSocket endpoint with the
          task status, result, and error information.
        - Prints a message to the console indicating the success or failure of the
          notification attempt.
    """
    go_app_url = os.getenv("GO_APP_URL", "http://localhost:3001/user-tasks/update/")
    payload = {
        "task_id": task_id,
        "status": status,
        "result": result,
        "error": error,
        "client_id": user_id,
        "organization_id": organization_id,
        "business_unit_id": business_unit_id,
    }

    print(f"Sending notification to Go app: {payload}")
    try:
        r = requests.post(go_app_url, json=payload)
        r.raise_for_status()
    except requests.RequestException as e:
        print(f"Failed to notify Go app: {e}")


@celery.task(name="generate_user_report", bind=True)
def generate_user_report(
    self,
    table_name: str,
    columns: List[str],
    relationships: Optional[List[Dict]],
    organization_id: str,
    business_unit_id: str,
    user_id: str,
    file_format: str,
    delivery_method: str,
):
    task_id = self.request.id
    payload = {
        "table_name": table_name,
        "columns": columns,
        "relationships": relationships,
        "organization_id": organization_id,
        "business_unit_id": business_unit_id,
        "file_format": file_format,
        "delivery_method": delivery_method,
    }

    with SessionLocal() as session:
        task_status = TaskStatus(
            task_id=task_id,
            status="RUNNING",
            organization_id=organization_id,
            business_unit_id=business_unit_id,
            user_id=user_id,
            payload=json.dumps(payload),  # Convert payload to JSON string
        )
        session.add(task_status)
        session.commit()

    try:
        # Convert relationships back to Relationship objects
        relationship_objects = None
        if relationships:
            relationship_objects = [Relationship(**rel) for rel in relationships]

        file_path = report_generate(
            table_name=table_name,
            columns=columns,
            relationships=relationship_objects,
            organization_id=organization_id,
            business_unit_id=business_unit_id,
            file_format=file_format,
            delivery_method=delivery_method,
        )

        with SessionLocal() as session:
            # Update the task status to 'SUCCESS' and log the file path
            stmt = (
                update(TaskStatus)
                .where(TaskStatus.task_id == task_id)
                .values(status="SUCCESS", result=json.dumps({"file_path": file_path}))
            )
            session.execute(statement=stmt)
            session.commit()

        notify_go_app(
            task_id=task_id,
            user_id=user_id,
            status="completed",
            result=file_path,
            organization_id=organization_id,
            business_unit_id=business_unit_id,
        )
    except Exception as e:
        with SessionLocal() as session:
            # Update the task status to 'FAILED' and log the error message
            stmt = (
                update(TaskStatus)
                .where(TaskStatus.task_id == task_id)
                .values(status="FAILED", result=json.dumps({"error": str(e)}))
            )
            session.execute(statement=stmt)
            session.commit()

        notify_go_app(
            task_id=task_id,
            user_id=user_id,
            status="failed",
            organization_id=organization_id,
            business_unit_id=business_unit_id,
            error=str(e),
        )
        raise e
