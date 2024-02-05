# --------------------------------------------------------------------------------------------------
#  COPYRIGHT(c) 2024 Trenova                                                                       -
#                                                                                                  -
#  This file is part of Trenova.                                                                   -
#                                                                                                  -
#  The Trenova software is licensed under the Business Source License 1.1. You are granted the right
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
import typing

from rest_framework import permissions, status, views, viewsets
from rest_framework.request import Request
from rest_framework.response import Response
from rest_framework.decorators import api_view, permission_classes

from core.permissions import CustomObjectPermissions
from integration import models, serializers
from integration.selectors import get_organization_google_api
from integration.services import autocomplete_location_service


class IntegrationVendorViewSet(viewsets.ModelViewSet):
    """A viewset for viewing and editing Integration Vendor information in the
    system.

    This viewset provides default operations for creating, updating, and
    deleting Integration Vendors, as well as listing and retrieving
    information. It uses the `IntegrationVendorSerializer` class to convert the
    integration vendor instances to and from JSON-formatted data.
    """

    queryset = models.IntegrationVendor.objects.all()
    serializer_class = serializers.IntegrationVendorSerializer
    permission_classes = [CustomObjectPermissions]


class IntegrationViewSet(viewsets.ModelViewSet):
    """A viewset for viewing and editing integration information in the system.

    The viewset provides default operations for creating, updating, and
    deleting customers, as well as listing and retrieving integrations. It uses
    the `IntegrationSerializer` class to convert the integration instances to
    and from JSON-formatted data.
    """

    queryset = models.Integration.objects.all()
    serializer_class = serializers.IntegrationSerializer
    permission_classes = [CustomObjectPermissions]


class GoogleAPIViewSet(viewsets.ModelViewSet):
    """A viewset for viewing and editing Google API keys in the system.

    The viewset provides default operations for creating, updating, and
    deleting Google API keys, as well as listing and retrieving Google API
    keys. It uses the `GoogleAPISerializer` class to convert the Google API key
    instances to and from JSON-formatted data.

    Only authenticated users and admins are allowed to access the views
    provided by this viewset.
    """

    queryset = models.GoogleAPI.objects.all()
    serializer_class = serializers.GoogleAPISerializer
    permission_classes = [CustomObjectPermissions]
    read_only_fields = ("organization", "business_unit")
    extra_kwargs = {
        "organization": {"required": False},
        "business_unit": {"required": False},
    }


class GoogleAPIDetailViewSet(views.APIView):
    """A viewset that gets the Google API details from the system for a
    specific organization.

    The viewset provides a GET operation for retrieving the Google API details
    for a specific organization. It uses the `GoogleAPISerializer` class to
    convert the Google API key instances to and from JSON-formatted data.

    Only authenticated users and admins are allowed to access the views
    provided by this viewset.
    """

    permission_classes = [permissions.IsAuthenticated]
    http_method_names = ["get", "options", "head", "put"]

    def get(
        self, request: Request, *args: typing.Any, **kwargs: typing.Any
    ) -> Response:
        user = request.user

        key_details = get_organization_google_api(organization=user.organization)  # type: ignore
        serializer = serializers.GoogleAPISerializer(key_details)

        return Response({"results": serializer.data}, status=status.HTTP_200_OK)

    def put(
        self, request: Request, *args: typing.Any, **kwargs: typing.Any
    ) -> Response:
        user = request.user

        key_details = get_organization_google_api(organization=user.organization)  # type: ignore
        serializer = serializers.GoogleAPISerializer(key_details, data=request.data)

        if not serializer.is_valid():
            return Response(serializer.errors, status=status.HTTP_400_BAD_REQUEST)

        serializer.save()
        return Response({"results": serializer.data}, status=status.HTTP_200_OK)


@api_view(["GET"])
@permission_classes([permissions.IsAuthenticated])
def autocomplete_location(request: Request) -> Response:
    """
    Autocomplete the location based on the search query.

    Args:
        request (Request): The request object.

    Returns:
        Response: A response object containing the location results.
    """
    search_query = request.query_params.get("search")
    if not search_query:
        return Response(
            {"error": "Search query is required."}, status=status.HTTP_400_BAD_REQUEST
        )

    organization = request.user.organization

    location_results = autocomplete_location_service(
        search_query=search_query, organization=organization
    )

    return Response(location_results, status=status.HTTP_200_OK)
