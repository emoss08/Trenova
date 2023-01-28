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

from billing import models, serializers
from utils.views import OrganizationMixin


class BillingControlViewSet(OrganizationMixin):
    """A viewset for viewing and editing BillingControl in the system.

    The viewset provides default operations for creating, updating Order Control,
    as well as listing and retrieving Order Control. It uses the ``BillingControlSerializer``
    class to convert the order control instance to and from JSON-formatted data.

    Only admin users are allowed to access the views provided by this viewset.

    Attributes:
        queryset (QuerySet): A queryset of BillingControl objects that will be used to
        retrieve and update BillingControl objects.

        serializer_class (BillingControlSerializer): A serializer class that will be used to
        convert BillingControl objects to and from JSON-formatted data.

        permission_classes (list): A list of permission classes that will be used to
        determine if a user has permission to perform a particular action.
    """

    queryset = models.BillingControl.objects.all()
    permission_classes = [permissions.IsAdminUser]
    serializer_class = serializers.BillingControlSerializer
    http_method_names = ["get", "put", "patch", "head", "options"]


class BillingQueueViewSet(OrganizationMixin):
    """
    A viewset for viewing and editing billing queue in the system.

    The viewset provides default operations for creating, updating, and deleting charge types,
    as well as listing and retrieving charge types. It uses the `BillingQueueSerializer` class to
    convert the charge type instances to and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this viewset.
    Filtering is also available, with the ability to filter by `order` pro_number, `worker` code, `customer`
    code, `revenue_code` code and `order_type` id.
    """

    queryset = models.BillingQueue.objects.all()
    serializer_class = serializers.BillingQueueSerializer
    filterset_fields = (
        "order__pro_number",
        "worker__code",
        "customer__code",
        "revenue_code__code",
        "order_type",
    )


class ChargeTypeViewSet(OrganizationMixin):
    """
    A viewset for viewing and editing charge types in the system.

    The viewset provides default operations for creating, updating, and deleting charge types,
    as well as listing and retrieving charge types. It uses the `ChargeTypeSerializer` class to
    convert the charge type instances to and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this viewset.
    Filtering is also available, with the ability to filter by charge type ID, name, and code.
    """

    queryset = models.ChargeType.objects.all()
    serializer_class = serializers.ChargeTypeSerializer
    filterset_fields = ("name",)


class AccessorialChargeViewSet(OrganizationMixin):
    """
    A viewset for viewing and editing accessorial charges in the system.

    The viewset provides default operations for creating, updating, and
    deleting accessorial charges, as well as listing and retrieving accessorial
    charges. It uses the `AccessorialChargeSerializer` class to convert the
    accessorial charge instances to and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this viewset.
    Filtering is also available, with the ability to filter by accessorial charge
    ID, code, and method.
    """

    queryset = models.AccessorialCharge.objects.all()
    serializer_class = serializers.AccessorialChargeSerializer
    filterset_fields = ("code", "is_detention", "method")


class DocumentClassificationViewSet(OrganizationMixin):
    """
    A viewset for viewing and editing document classifications in the system.

    The viewset provides default operations for creating, updating, and
    deleting document classifications, as well as listing and retrieving document classifications.
    It uses the `DocumentClassificationSerializer`
    class to convert the document classification instances to and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this viewset.
    Filtering is also available, with the ability to filter by document classification
    ID, and name.
    """

    queryset = models.DocumentClassification.objects.all()
    serializer_class = serializers.DocumentClassificationSerializer
    filterset_fields = ("name",)
