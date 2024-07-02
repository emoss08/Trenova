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
