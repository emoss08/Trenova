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

from celery import shared_task

from order.models import Order
from order.services.consolidate_pdf import combine_pdfs


@shared_task  # type: ignore
def consolidate_order_documentation(order_id: str) -> None:
    """Consolidate Order

    Query the database for the Order and call the consolidate_pdf
    service to combine the PDFs into a single PDF.

    Args:
        order_id (str): Order ID

    Returns:
        None
    """

    order: Order = Order.objects.get(id=order_id)
    combine_pdfs(order=order)
