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
from typing import Dict, List

import requests

GITHUB_REPO_API_URL = (
    "https://api.github.com/repos/Monta-Application/monta-plugins/contents"
)


def get_plugin_list() -> List[Dict[str, str]]:
    response = requests.get(GITHUB_REPO_API_URL)
    response_data = response.json()
    plugin_list = []

    for plugin in response_data:
        if plugin["name"].endswith(".zip"):
            plugin_name = plugin["name"].replace(".zip", "")
            plugin_list.append(
                {"name": plugin_name, "download_url": plugin["download_url"]}
            )

    return plugin_list