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

from movements import serializers, models
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
        "equipment",
        "primary_worker",
        "secondary_worker",
    )