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

import httpx

from integration.utils import IntegrationBase


class IntegrationAuthenticationService(IntegrationBase):
    """
    Base class for Integration Authentication Services
    """

    async def authenticate(self) -> str:
        """Authenticate the integration with provided credentials
        and return the token.

        Returns:
            str: Token
        """
        async with httpx.AsyncClient() as client:
            response = await client.post(
                self.integration.login_url,
                data={
                    "username": self.integration.username,
                    "password": self.integration.password,
                },
            )
            response.raise_for_status()
            return response.json()["token"]

    async def set_token(self) -> None:
        """Set the new token in the database.

        Returns:
            None
        """
        self.integration.token = await self.authenticate()
        self.integration.save()

    async def launch(self) -> None:
        """Launch the authentication service.

        Returns:
            None
        """
        await self.set_token()
