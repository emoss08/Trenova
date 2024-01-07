# --------------------------------------------------------------------------------------------------
#  COPYRIGHT(c) 2024 MONTA                                                                         -
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
import logging

from channels.db import database_sync_to_async
from channels.exceptions import DenyConnection
from channels.generic.websocket import AsyncJsonWebsocketConsumer
from channels.layers import get_channel_layer

channel_layer = get_channel_layer()


class NotificationConsumer(AsyncJsonWebsocketConsumer):
    """
    A consumer for notifications, handling user authentication, group management, and notification sending.
    """

    async def connect(self):
        """
        Handles the WebSocket connection process. Authenticates the user and adds them to a group.
        Rejects the connection if authentication fails.
        """
        try:
            user = self.scope["user"]
            if not user.is_authenticated:
                raise DenyConnection("User is not authenticated.")

            self.room_group_name = await self.get_username(user)
            await self.channel_layer.group_add(self.room_group_name, self.channel_name)
            await self.accept()
        except DenyConnection as e:
            logging.error(f"Connection denied: {e}")
            await self.close()

    async def disconnect(self, close_code):
        """
        Handles the WebSocket disconnection process. Removes the user from their group.
        """
        try:
            await self.channel_layer.group_discard(
                self.room_group_name, self.channel_name
            )
        except Exception as e:
            logging.error(f"Error on disconnect: {e}")

    async def send_notification(self, event):
        """
        Sends a notification to the client. Expects an event in the format {'type': x, 'value': y}.
        """
        try:
            await self.send_json(event)
        except Exception as e:
            logging.error(f"Error sending notification: {e}")

    @database_sync_to_async
    def get_username(self, user):
        """
        Asynchronously retrieves the username of a user.
        """
        return user.username
