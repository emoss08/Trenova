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

from rest_framework import permissions

from invoicing import models, serializers
from utils.views import OrganizationMixin


class InvoiceControlViewSet(OrganizationMixin):
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
