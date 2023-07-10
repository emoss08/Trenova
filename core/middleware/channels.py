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
from http.cookies import SimpleCookie

from channels.db import database_sync_to_async

from accounts.authentication import BearerTokenAuthentication


class TokenAuthMiddleware:
    def __init__(self, app) -> None:
        self.app = app

    async def __call__(self, scope, receive, send) -> None:
        token_authenticator = BearerTokenAuthentication()

        headers = dict(scope["headers"])
        cookies = SimpleCookie()
        cookies.load(headers.get(b"cookie", b"").decode())

        token = cookies.get("auth_token")
        token = token.value if token else None

        if token is None:
            return await self.close(send)

        mock_request = type("", (), {})()
        mock_request.META = {"HTTP_AUTHORIZATION": f"Bearer {token}"}

        user_token = await database_sync_to_async(token_authenticator.authenticate)(
            mock_request
        )

        if user_token is None:
            return await self.close(send)

        scope["user"] = user_token[0]
        return await self.app(scope, receive, send)

    async def close(self, send) -> None:
        await send(
            {
                "type": "websocket.close",
                "code": 1000,
                "reason": "Authentication failed",
            }
        )
