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

from django.db.models import QuerySet
from rest_framework import viewsets

from commodities import models, serializers
from core.permissions import CustomObjectPermissions


class HazardousMaterialViewSet(viewsets.ModelViewSet):
    """A viewset for viewing and editing hazardous materials in the system.

    The viewset provides default operations for creating, updating, and deleting hazardous materials,
    as well as listing and retrieving hazardous materials. It uses the `HazardousMaterialSerializer`
    class to convert the hazardous material instances to and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this viewset.
    Filtering is also available, with the ability to filter by hazardous material ID, name, and
    description.
    """

    queryset = models.HazardousMaterial.objects.all()
    serializer_class = serializers.HazardousMaterialSerializer
    filterset_fields = (
        "status",
        "name",
    )
    permission_classes = [CustomObjectPermissions]
    search_fields = ("name", "status", "erg_number", "proper_shipping_name")

    def get_queryset(self) -> QuerySet[models.HazardousMaterial]:
        queryset: QuerySet[models.HazardousMaterial] = self.queryset.filter(
            organization_id=self.request.user.organization_id  # type: ignore
        ).only(
            "id",
            "packing_group",
            "hazard_class",
            "organization_id",
            "name",
            "description",
            "status",
            "erg_number",
            "proper_shipping_name",
        )
        return queryset


class CommodityViewSet(viewsets.ModelViewSet):
    """A viewset for viewing and editing commodities in the system.

    The viewset provides default operations for creating, updating, and deleting commodities,
    as well as listing and retrieving commodities. It uses the `CommoditySerializer`
    class to convert the commodity instances to and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this viewset.
    Filtering is also available, with the ability to filter by commodity ID, name, and
    description.
    """

    queryset = models.Commodity.objects.all()
    serializer_class = serializers.CommoditySerializer
    filterset_fields = ("status", "name")
    permission_classes = [CustomObjectPermissions]

    def get_queryset(self) -> QuerySet[models.Commodity]:
        """
        Returns the queryset of commodities that are associated with the current user's organization.

        Returns:
            The queryset of commodities that are associated with the current user's organization.
        """
        queryset = self.queryset.filter(
            organization_id=self.request.user.organization_id  # type: ignore
        ).only(
            "id",
            "min_temp",
            "organization_id",
            "is_hazmat",
            "name",
            "description",
            "set_point_temp",
            "unit_of_measure",
            "hazardous_material__id",
            "max_temp",
            "created",
            "modified",
        )

        return queryset
