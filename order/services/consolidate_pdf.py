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

from django.conf import settings
from django.core.files.storage import default_storage
from pypdf import PdfMerger

from billing.models import DocumentClassification
from order.models import Order, OrderDocumentation


def combine_pdfs_service(*, order: Order) -> OrderDocumentation:
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
