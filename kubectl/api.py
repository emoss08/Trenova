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
from django.shortcuts import get_object_or_404
from rest_framework import status
from rest_framework.decorators import api_view
from rest_framework.request import Request
from rest_framework.response import Response

from accounts.models import User
from kubectl.helpers import (
    get_node_info,
    get_node_metadata,
    organization_kube_api_client,
)


@api_view(["GET"])
def get_active_clusters(request: Request) -> Response:
    """Handles GET requests to retrieve information about active clusters from the organization's
     Kubernetes API.

    Args:
        request(Request): A HTTP request object containing metadata about the client's request.

    Returns:
        Response (Response): A HTTP response object containing a list of dictionaries, where each dictionary contains information
        about an active cluster, including the cluster's name, node information, and metadata.
    """
    user = get_object_or_404(User, username=request.user.username)

    api = organization_kube_api_client(organization=user.organization)
    node = api.list_node()

    response = [
        {
            "name": node.metadata.name,
            "node_info": get_node_info(node=node),
            "metadata": get_node_metadata(node=node),
        }
        for node in node.items
    ]
    return Response(response, status=status.HTTP_200_OK)
