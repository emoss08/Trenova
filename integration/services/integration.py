# -*- coding: utf-8 -*-
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

from typing import Optional

import requests

from ..models import Integration, IntegrationAuthTypes


class IntegrationServices(object):
    """
    Service Class for the integration application.
    """

    @staticmethod
    def get_integration(integration_id: int) -> Integration:
        """Get Integration by integration_id.

        Args:
            integration_id (int): parameter to capture the primary key of the integration.

        Returns:
            Integration: Returns the integration object.

        Typical usage example:
            >>> integration_services = IntegrationServices().get_integration(integration_id=1)
        """
        return Integration.objects.get(id__exact=integration_id)

    def re_authenticate_integration(self, integration_id: int) -> Optional[Integration]:
        """Re-authenticate Integration by integration_id.

        Args:
            integration_id (int): parameter to capture the primary key of the integration.

        Returns:
            Optional[Integration]: Returns the integration object.

        Raises:
            ValidationError: Raises a validation error if the integration is not found.

        Typical usage example:
            >>> integration_services = IntegrationServices().re_authenticate_integration(integration_id=1)
        """
        obj: Integration = self.get_integration(integration_id=integration_id)

        if obj.auth_type == IntegrationAuthTypes.BEARER_TOKEN:
            try:
                response: requests.Response = requests.post(
                    url=obj.login_url,
                    headers={
                        "Content-Type": "application/json",
                        "Accept": "application/json",
                        "Authorization": f"Bearer {obj.auth_token}",
                    },
                )
                response.raise_for_status()
            except* requests.exceptions.HTTPError as err:
                raise err
            except* requests.exceptions.RequestException as err:
                raise err
            except* Exception as err:
                raise err

            json_obj = response.json()
            obj.auth_token = json_obj["token"]
            obj.save()
            return obj
        return None
