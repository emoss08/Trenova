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

from rest_framework import permissions, viewsets

from invoicing import models, serializers

if TYPE_CHECKING:
    from django.db.models import QuerySet


class InvoiceControlViewSet(viewsets.ModelViewSet):
    """A viewset for viewing and editing InvoiceControl in the system.

    The viewset provides default operations for creating, updating Order Control,
    as well as listing and retrieving Order Control. It uses the ``InvoiceControlSerializer``
    class to convert the order control instance to and from JSON-formatted data.

    Only admin users are allowed to access the views provided by this viewset.

    Attributes:
        queryset (QuerySet): A queryset of InvoiceControl objects that will be used to
        retrieve and update InvoiceControl objects.

        serializer_class (InvoiceControlSerializer): A serializer class that will be used to
        convert InvoiceControl objects to and from JSON-formatted data.

        permission_classes (list): A list of permission classes that will be used to
        determine if a user has permission to perform a particular action.
    """

    queryset = models.InvoiceControl.objects.all()
    permission_classes = [permissions.IsAdminUser]
    serializer_class = serializers.InvoiceControlSerializer
    http_method_names = ["get", "put", "patch", "head", "options"]

    def get_queryset(self) -> "QuerySet[models.InvoiceControl]":
        queryset = self.queryset.filter(
            organization_id=self.request.user.organization_id  # type: ignore
        ).only(
            "id",
            "invoice_number_prefix",
            "invoice_footer",
            "credit_memo_number_prefix",
            "attach_pdf",
            "invoice_logo_width",
            "invoice_terms",
            "invoice_logo",
            "show_amount_due",
            "show_invoice_due_date",
            "invoice_date_format",
            "id",
            "invoice_due_after_days",
            "organization_id",
        )
        return queryset
