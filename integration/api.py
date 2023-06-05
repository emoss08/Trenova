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

from rest_framework import viewsets

from integration import models, serializers


class IntegrationVendorViewSet(viewsets.ModelViewSet):
    """A viewset for viewing and editing Integration Vendor information in the system.

    This viewset provides default operations for creating, updating, and deleting Integration
    Vendors, as well as listing and retrieving information. It uses the `IntegrationVendorSerializer`
    class to convert the integration vendor instances to and from JSON-formatted data.
    """

    queryset = models.IntegrationVendor.objects.all()
    serializer_class = serializers.IntegrationVendorSerializer


class IntegrationViewSet(viewsets.ModelViewSet):
    """A viewset for viewing and editing integration information in the system.

    The viewset provides default operations for creating, updating, and deleting customers,
    as well as listing and retrieving integrations. It uses the `IntegrationSerializer`
    class to convert the integration instances to and from JSON-formatted data.
    """

    queryset = models.Integration.objects.all()
    serializer_class = serializers.IntegrationSerializer


class GoogleAPIViewSet(viewsets.ModelViewSet):
    """A viewset for viewing and editing Google API keys in the system.

    The viewset provides default operations for creating, updating, and deleting Google API keys,
    as well as listing and retrieving Google API keys. It uses the `GoogleAPISerializer`
    class to convert the Google API key instances to and from JSON-formatted data.

    Only authenticated users and admins are allowed to access the views provided by this viewset.
    """

    queryset = models.GoogleAPI.objects.all()
    serializer_class = serializers.GoogleAPISerializer
