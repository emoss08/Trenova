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

from control_file import models, serializers
from utils.views import OrganizationViewSet


class GoogleAPIViewSet(OrganizationViewSet):
    """A viewset for viewing and editing Google API keys in the system.

    The viewset provides default operations for creating, updating, and deleting Google API keys,
    as well as listing and retrieving Google API keys. It uses the `GoogleAPISerializer`
    class to convert the Google API key instances to and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this viewset.
    Filtering is also available, with the ability to filter by Google API key ID, name, and
    description.
    """

    queryset = models.GoogleAPI.objects.all()
    serializer_class = serializers.GoogleAPISerializer
    permission_classes = (permissions.IsAuthenticated, permissions.IsAdminUser)
