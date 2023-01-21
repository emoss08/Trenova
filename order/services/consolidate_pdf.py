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

from django.conf import settings
from django.core.files.storage import default_storage
from pypdf import PdfMerger

from billing.models import DocumentClassification
from order.models import Order, OrderDocumentation


def combine_pdfs(*, order: Order) -> OrderDocumentation:
    """Combine all PDFs in Order Document into one PDF file

    Args:
        order (Order): Order to combine documents from

    Returns:
        OrderDocumentation: created OrderDocumentation
    """

    document_class: DocumentClassification = DocumentClassification.objects.get(
        name="CON"
    )
    file_path = f"{settings.MEDIA_ROOT}/{order.id}.pdf"
    merger = PdfMerger()

    if default_storage.exists(file_path):
        raise FileExistsError(f"File {file_path} already exists")

    for document in order.order_documentation.all():
        merger.append(document.document.path)

    merger.write(file_path)
    merger.close()

    consolidated_document = default_storage.open(file_path, "rb")

    documentation: OrderDocumentation = OrderDocumentation.objects.create(
        organization=order.organization,
        order=order,
        document=consolidated_document,
        document_class=document_class,
    )

    default_storage.delete(file_path)

    return documentation
