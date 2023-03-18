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

from movements import models, serializers
from utils.views import OrganizationMixin


class MovementViewSet(OrganizationMixin):
    """A viewset for viewing and editing Movement information in the system.

    The viewset provides default operations for creating, updating and deleting movements,
    as well as listing and retrieving movements. It uses `MovementSerializer` class to
    convert the movement instances and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this viewset.
    Filter is also available, with the ability to filter by equipment, primary_worker and
    secondary_worker.

    Attributes:
        queryset (models.Movement): The queryset of the viewset.
        serializer_class (serializers.MovementSerializer): The serializer class of the viewset.
    """

    queryset = models.Movement.objects.all()
    serializer_class = serializers.MovementSerializer
    filterset_fields = (
        "tractor",
        "primary_worker__code",
        "secondary_worker__code",
        "order__pro_number"
    )
