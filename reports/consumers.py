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

from asgiref.sync import sync_to_async
from channels.db import database_sync_to_async
from channels.generic.websocket import AsyncJsonWebsocketConsumer
from channels.layers import get_channel_layer

from accounts.authentication import BearerTokenAuthentication

channel_layer = get_channel_layer()


class NotificationConsumer(AsyncJsonWebsocketConsumer):
    """This class inherits from JsonWebsocketConsumer and serves as a consumer for notifications.

    The NotificationConsumer class consumes notifications, authenticates the user from a provided token,
    adds the user to a communication group, and sends notifications to the user.

    Attributes:
        scope: A dictionary-like object that contains metadata about the connection.
        channel_name: A unique channel name automatically set on the base AsyncConsumer when a new
        connection is accepted.

    Methods:
        connect: Handles the connection process for a new consumer. It authenticates the user,
        adds the user to a group, and accepts the connection.
        disconnect: Handles the disconnection process for the consumer. It removes the user from the group.
        send_notification: Sends the notification data to the user/client as JSON.
    """

    async def connect(self) -> None:
        """This method is called when the websocket is handshaking as part of the connection process.

        The method authenticates the user using a bearer token obtained from cookies, adds the user to
        a group (named by the user's username), and accepts the incoming socket connection.

        If the token is not provided or authentication fails, the method returns without doing anything.

        Returns:
            None: This function does not return anything.

        Raises:
            HTTPError: If the token_authenticator.authenticate() fails to authenticate.
        """

        token_authenticator = BearerTokenAuthentication()

        headers = dict(self.scope["headers"])
        cookies = SimpleCookie()
        cookies.load(headers.get(b"cookie", b"").decode())

        token = cookies.get("auth_token")
        token = token.value if token else None

        if token is None:
            return

        mock_request = type("", (), {})()
        mock_request.META = {"HTTP_AUTHORIZATION": f"Bearer {token}"}

        user_token = await database_sync_to_async(token_authenticator.authenticate)(
            mock_request
        )
        if user_token is None:
            return

        self.scope["user"] = user_token[0]
        self.room_group_name = await sync_to_async(self.scope["user"].get_username)()
        await self.channel_layer.group_add(
            self.room_group_name, self.channel_name
        )
        await self.accept()

    async def disconnect(self, close_code: int) -> None:
        """This method is called when the WebSocket closes for any reason.

        The method removes the user from the group.

        Args:
            close_code (int): An integer that provides more detail on why the connection was closed.

        Returns:
            None: This function does not return anything.
        """
        await self.channel_layer.group_discard(self.room_group_name, self.channel_name)

    async def send_notification(self, event: dict):
        """Sends the notification data to the client as JSON.

        The event dict should follow the format {type: x, value: y}, where 'type' is the type of
        event and 'value' is the data related to the event.

        Args:
            event (dict): A dictionary containing the event data to be sent.

        Returns:
            None: This function does not return anything.
        """
        await self.send_json(event)
