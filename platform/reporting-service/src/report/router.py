# COPYRIGHT(c) 2024 Trenova
#
# This file is part of Trenova.
#
# The Trenova software is licensed under the Business Source License 1.1. You are granted the right
# to copy, modify, and redistribute the software, but only for non-production use or with a total
# of less than three server instances. Starting from the Change Date (November 16, 2026), the
# software will be made available under version 2 or later of the GNU General Public License.
# If you use the software in violation of this license, your rights under the license will be
# terminated automatically. The software is provided "as is," and the Licensor disclaims all
# warranties and conditions. If you use this license's text or the "Business Source License" name
# and trademark, you must comply with the Licensor's covenants, which include specifying the
# Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
# Grant, and not modifying the license in any other way.


from celery.result import AsyncResult
from fastapi import APIRouter
from fastapi.responses import JSONResponse

from report.schemas import Report
from report.service import report_generate
from worker import generate_user_report

router = APIRouter()


@router.post("/generate-report/")
def generate_report(report: Report):
    # Convert Pydantic model to dictionary
    report_dict = report.model_dump()

    # Pass the dictionary to the Celery task
    task = generate_user_report.delay(
        table_name=report_dict["tableName"],
        columns=report_dict["columns"],
        relationships=report_dict["relationships"],
        organization_id=report_dict["organizationId"],
        business_unit_id=report_dict["businessUnitId"],
        file_format=report_dict["fileFormat"],
        delivery_method=report_dict["deliveryMethod"],
        user_id=report_dict["userId"],
    )

    return JSONResponse({"task_id": task.id})


@router.get("/get-report/{task_id}")
def get_user_report(task_id: str) -> JSONResponse:
    task_result = AsyncResult(task_id, app=generate_user_report)

    result = {
        "task_id": task_result.id,
        "task_status": task_result.status,
        "task_result": task_result.result,
    }

    return JSONResponse(result)
