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

from equipment import models, serializers
from utils.views import OrganizationMixin


class EquipmentTypeViewSet(OrganizationMixin):
    """A viewset for viewing and editing customer information in the system.

    The viewset provides default operations for creating, updating, and deleting customers,
    as well as listing and retrieving customers. It uses the `CustomerSerializer`
    class to convert the customer instances to and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this viewset.
    Filtering is also available, with the ability to filter by customer ID, name, and
    code.
    """

    queryset = models.EquipmentType.objects.all()
    serializer_class = serializers.EquipmentTypeSerializer


class TractorViewSet(OrganizationMixin):
    """A viewset for viewing and editing tractors information in the system.

    The viewset provides default operations for creating, updating, and deleting tractors,
    as well as listing and retrieving tractors. It uses the `TractorSerializer`
    class to convert the tractor instances to and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this viewset.
    Filtering is also available, with the ability to filter by is_active, manufacturer,
    has_berth, equipment_type__name, fleet__code and highway_use_tax.
    """

    queryset = models.Tractor.objects.all()
    serializer_class = serializers.TractorSerializer
    filterset_fields = (
        "is_active",
        "manufacturer",
        "equipment_type__name",
        "fleet__code",
        "has_berth",
        "highway_use_tax",
    )

class TrailerViewSet(OrganizationMixin):
    """A viewset for viewing and editing customer information in the system.

    The viewset provides default operations for creating, updating, and deleting trailers,
    as well as listing and retrieving trailers. It uses the `Trailers`
    class to convert the customer instances to and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this viewset.
    Filtering is also available, with the ability to filter by is_active, equipment_type__name,
    fleet_code__code, and is_leased.
    """

    queryset = models.Trailer.objects.all()
    serializer_class = serializers.TrailerSerializer
    filterset_fields = (
        "is_active",
        "equipment_type__name",
        "fleet__code",
        "is_leased"
    )


class EquipmentManufacturerViewSet(OrganizationMixin):
    """A viewset for viewing and editing customer information in the system.

    The viewset provides default operations for creating, updating, and deleting customers,
    as well as listing and retrieving customers. It uses the `CustomerSerializer`
    class to convert the customer instances to and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this viewset.
    Filtering is also available, with the ability to filter by customer ID, name, and
    code.
    """

    queryset = models.EquipmentManufacturer.objects.all()
    serializer_class = serializers.EquipmentManufacturerSerializer


class EquipmentMaintenancePlanViewSet(OrganizationMixin):
    """A viewset for viewing and editing customer information in the system.

    The viewset provides default operations for creating, updating, and deleting customers,
    as well as listing and retrieving customers. It uses the `CustomerSerializer`
    class to convert the customer instances to and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this viewset.
    Filtering is also available, with the ability to filter by customer ID, name, and
    code.
    """

    queryset = models.EquipmentMaintenancePlan.objects.all()
    serializer_class = serializers.EquipmentMaintenancePlanSerializer
