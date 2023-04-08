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

from rest_framework import permissions

from dispatch import models, serializers
from utils.views import OrganizationMixin


class CommentTypeViewSet(OrganizationMixin):
    """A viewset for viewing and editing Comment Type information in the system.

    The viewset provides default operations for creating, updating, and deleting Comment
    Types, as well as listing and retrieving Comment Types. It uses the `CommentTypeSerializer`
    class to convert the customer instances to and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this viewset.
    """

    queryset = models.CommentType.objects.all()
    serializer_class = serializers.CommentTypeSerializer


class DelayCodeViewSet(OrganizationMixin):
    """A viewset for viewing and editing Delay Code information in the system.

    The viewset provides default operations for creating, updating, and deleting Delay
    Codes, as well as listing and retrieving Delay Codes. It uses the `DelayCodeSerializer`
    class to convert the customer instances to and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this viewset.
    """

    queryset = models.DelayCode.objects.all()
    serializer_class = serializers.DelayCodeSerializer


class FleetCodeViewSet(OrganizationMixin):
    """A viewset for viewing and editing Fleet Code information in the system.

    The viewset provides default operations for creating, updating, and deleting Fleet Codes,
    as well as listing and retrieving Fleet Codes. It uses the `FleetCodeSerializer`
    class to convert the Fleet Codes instances to and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this viewset.
    Filtering is also available, with the ability to filter by is active.
    """

    queryset = models.FleetCode.objects.all()
    serializer_class = serializers.FleetCodeSerializer
    filterset_fields = ("is_active",)


class DispatchControlViewSet(OrganizationMixin):
    """A viewset for viewing and editing Dispatch Control in the system.

    The viewset provides default operations for updating, as well as listing and retrieving
    Dispatch Control. It uses the `DispatchControlSerializer` class to convert the Dispatch
    Control instances to and from JSON-formatted data.

    Only get, put, patch, head and options HTTP methods are allowed when using this viewset.
    Only Admin users are allowed to access the views provided by this viewset.
    """

    queryset = models.DispatchControl.objects.all()
    serializer_class = serializers.DispatchControlSerializer
    permission_classes = [permissions.IsAdminUser]
    http_method_names = ["get", "put", "patch", "head", "options"]


class RateViewSet(OrganizationMixin):
    """A viewset for viewing and editing Rate information in the system.

    The viewset provides default operations for creating, updating, and deleting Rates,
    as well as listing and retrieving Rates. It uses the `RateSerializer`
    class to convert the Rate instances to and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this viewset.
    Filtering is also available, with the ability to filter by is active.
    """

    queryset = models.Rate.objects.all()
    serializer_class = serializers.RateSerializer

class RateBillingTableViewSet(OrganizationMixin):
    """
    Django Rest Framework ViewSet for the RateBillingTable model.

    The RateBillingTableViewSet class provides the CRUD operation for the RateBillingTable model
    through the Django Rest Framework. The class is a subclass of OrganizationMixin,
    which provides the organization-related functionality.

    Attributes:
        queryset (models.RateBillingTable.objects.all()): The default queryset for the viewset.
        serializer_class (serializers.RateBillingTableSerializer): The serializer class for the viewset.
        filterset_fields (tuple): The fields to use for filtering the queryset.
    """

    queryset = models.RateBillingTable.objects.all()
    serializer_class = serializers.RateBillingTableSerializer
    filterset_fields = ("rate",)
