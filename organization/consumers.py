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

import json
from typing import Any, Optional, Union

from channels.generic.websocket import AsyncWebsocketConsumer


class KeepAliveConsumer(AsyncWebsocketConsumer):
    """Django Channels Consumer for handling Keep-Alive messages.

    This consumer accepts WebSocket connections and listens for Keep-Alive messages
    sent by clients to the "keepalive" group. When a message is received, it is broadcast
    to all the clients in the same group.

    Methods:
        connect: Connects the client to the WebSocket and adds it to the "keepalive" group.
        disconnect: Disconnects the client from the WebSocket and removes it from the "keepalive" group.
        receive: Receives messages from the client and broadcasts them to the "keepalive" group.
        keepalive_message: Receives messages broadcast to the "keepalive" group and sends them to the client.
    """

    async def connect(self) -> None:
        """Connects the client to the WebSocket and adds it to the "keepalive" group.

        Returns:
            None: This method does not return anything.
        """
        self.user = self.scope["user"]
        print(self.user)
        await self.accept()
        await self.channel_layer.group_add("keepalive", self.channel_name)

    async def disconnect(self, close_code: str) -> None:
        """Disconnects the client from the WebSocket and removes it from the "keepalive" group.

        Args:
            close_code (str): The close code sent by the client when disconnecting.

        Returns:
            None: This method does not return anything.
        """
        await self.channel_layer.group_discard("keepalive", self.channel_name)

    async def receive(
        self,
        text_data: str | bytes | None = None,
        bytes_data: bytearray | None = None,
    ) -> None:
        """Receives messages from the client and broadcasts them to the "keepalive" group.

        Args:
            bytes_data (bytearray, optional): The binary data sent by the client.
            text_data (str | bytes, optional): The text data sent by the client.

        Returns:
            None: This method does not return anything.
        """
        if text_data is not None:
            keep_alive_data = json.loads(text_data)
            message = keep_alive_data["message"]

            await self.channel_layer.group_send(
                "keepalive", {"type": "keepalive_message", "message": message}
            )

    async def keepalive_message(self, event: dict[str, Any]) -> None:
        """Receives messages broadcast to the "keepalive" group and sends them to the client.

        Args:
            event (dict[str, Any]): A dictionary containing the message sent by the client.

        Returns:
            None
        """
        message = event["message"]
        await self.send(text_data=json.dumps({"message": message}))
