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
from asgiref.sync import async_to_sync
from channels.generic.websocket import JsonWebsocketConsumer
from channels.layers import get_channel_layer

from accounts.authentication import BearerTokenAuthentication

channel_layer = get_channel_layer()


class NotificationConsumer(JsonWebsocketConsumer):
    def connect(self):
        token_authenticator = BearerTokenAuthentication()

        # Get the token from the URL
        token = self.scope["query_string"].decode("utf-8").split("=")[1]

        # Mocking a request to verify the token
        mock_request = type("", (), {})()  # Create a blank class
        mock_request.META = {"HTTP_AUTHORIZATION": f"Bearer {token}"}

        user_token = token_authenticator.authenticate(mock_request)
        if user_token is None:
            return

        self.scope["user"] = user_token[0]

        self.room_group_name = self.scope["user"].username
        async_to_sync(channel_layer.group_add)(self.room_group_name, self.channel_name)
        self.accept()

    def disconnect(self, close_code):
        async_to_sync(channel_layer.group_discard)(
            self.room_group_name, self.channel_name
        )

    def send_notification(self, event):
        self.send_json(event)
