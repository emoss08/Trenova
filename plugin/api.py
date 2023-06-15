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

from django.core.management import call_command
from rest_framework import status
from rest_framework.decorators import api_view
from rest_framework.request import Request
from rest_framework.response import Response

from plugin import serializers, utils


@api_view(["GET"])
def get_plugin_list_api(request: Request) -> Response:
    plugin_list = utils.get_plugin_list()
    serializer = serializers.PluginInfoSerializer(plugin_list, many=True)
    return Response(serializer.data)


# TODO: Change this to POST METHOD, and body should be a JSON with the plugin name
@api_view(["GET"])
def plugin_install_api(request: Request) -> Response:
    plugin_name = request.query_params.get("plugin_name", None)

    if plugin_name is None:
        return Response(
            {"detail": "Missing 'plugin_name' query parameter."},
            status=status.HTTP_400_BAD_REQUEST,
        )

    plugin_list = utils.get_plugin_list()
    plugin = next((p for p in plugin_list if p["name"] == plugin_name), None)

    if plugin is None:
        return Response(
            {"detail": "Plugin not found"}, status=status.HTTP_404_NOT_FOUND
        )

    # Run the install_plugin command
    call_command("install_plugin", plugin["download_url"])

    return Response(
        {"detail": "Plugin installed successfully"}, status=status.HTTP_200_OK
    )
