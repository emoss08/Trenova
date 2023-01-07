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
from django_filters.rest_framework import DjangoFilterBackend
from rest_framework import permissions

from billing import models, serializers
from utils.views import OrganizationViewSet


class ChargeTypeViewSet(OrganizationViewSet):
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
    permission_classes = (permissions.IsAuthenticated,)
    filter_backends = (DjangoFilterBackend,)
    filterset_fields = ("id", "name")


class AccessorialChargeViewSet(OrganizationViewSet):
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
    permission_classes = (permissions.IsAuthenticated,)
    filter_backends = (DjangoFilterBackend,)
    filterset_fields = ("code", "is_detention", "charge_amount", "method")


class DocumentClassificationViewSet(OrganizationViewSet):
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
    permission_classes = (permissions.IsAuthenticated,)
    filter_backends = (DjangoFilterBackend,)
    filterset_fields = (
        "id",
        "name",
    )
