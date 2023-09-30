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
from typing import TYPE_CHECKING

from backend.celery import app
from core.exceptions import ServiceException
from shipment import selectors, services
from utils.types import ModelUUID

if TYPE_CHECKING:
    from celery.app.task import Task


@app.task(
    name="consolidate_shipment_documentation",
    bind=True,
    max_retries=3,
    default_retry_delay=60,
    queue="medium_priority",
)
def consolidate_shipment_documentation(self: "Task", *, shipment_id: ModelUUID) -> None:
    """Consolidate Order

    Query the database for the shipment and call the consolidate_pdf
    service to combine the PDFs into a single PDF.

    Args:
        self (celery.app.task.Task): The task object
        shipment_id (str): shipment ID

    Returns:
        None: None

    Raises:
        ObjectDoesNotExist: If the shipment does not exist in the database.
    """

    try:
        if shipment := selectors.get_shipment_by_id(shipment_id=shipment_id):
            services.combine_pdfs_service(shipment=shipment)
        else:
            return None

    except ServiceException as exc:
        raise self.retry(exc=exc) from exc
