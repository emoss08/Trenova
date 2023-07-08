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

"""
A client for interacting with the billing API.

Steps:

1. Show orders ready to be billed based on BillingControl Transfer Client. (Send payload from websocket to client.)
2. Allow user to select orders to be transferred to billing queue. (Receive payload from client.)
3. Show dropdown of missing billing criteria for a specific customer (Highlight Red).
4. Show what orders are able to be billed (Highlight Green).
5. Navigate user to Billing Queue page. (Send API Response to client.)
6. Allow user to select orders to be billed. (Receive payload from client.)
7. Send payload to billing API.
8. Receive response from billing API.
9. Send response to client.
10. Update orders in database.
11. Update billing queue in database.
12. Update billing transfer log in database.
13. Update billing exceptions in database.
"""
import uuid
from asyncio import sleep
from typing import Any

from asgiref.sync import sync_to_async
from channels.db import database_sync_to_async
from channels.generic.websocket import AsyncJsonWebsocketConsumer
from channels.layers import get_channel_layer
from billing.selectors import get_invoices_by_invoice_number
from billing.services import (
    transfer_to_billing_queue_service,
    bill_orders,
    BillingClientSessionManager,
)

channel_layer = get_channel_layer()


class BillingClientConsumer(AsyncJsonWebsocketConsumer):
    def __init__(self, *args: Any, **kwargs: Any):
        super().__init__()
        self.session_manager = None
        self.session = None
        self.room_group_name = None

    async def connect(self) -> None:
        self.session_manager = BillingClientSessionManager()
        self.room_group_name = await sync_to_async(self.scope["user"].get_username)()

        self.session = await sync_to_async(
            self.session_manager.set_new_billing_client_session
        )(self.scope["user"].id)
        await self.channel_layer.group_add(self.room_group_name, self.channel_name)
        await self.accept()

    async def disconnect(self, close_code: int) -> None:
        # Delete the session when the client disconnects
        if hasattr(self, "session_manager") and hasattr(
            self.scope, "user"
        ):  # Check if session_manager exists
            await sync_to_async(self.session_manager.delete_billing_client_session)(
                self.scope["user"].id
            )

        if hasattr(self, "room_group_name") and hasattr(
            self, "channel_name"
        ):  # Check if room_group_name and channel_name exist
            await self.channel_layer.group_discard(
                self.room_group_name, self.channel_name
            )

    async def receive_json(self, content: dict[str, Any], **kwargs: Any) -> None:
        action = content.get("action")

        # Map actions to their corresponding methods
        action_map = {
            "restart": self.get_started,
            "get_started": self.get_started,
            "orders_ready": self.send_to_billing_queue,
            "billing_queue": self.send_to_billing_queue,
            "bill_orders": self.bill_orders,
        }

        # Save the previous action and last payload before processing the new action
        self.session["previous_action"] = self.session["current_action"]
        self.session["last_payload"] = content
        self.session["current_action"] = action

        if action in action_map:
            await action_map[action](content)

            # After processing the action, update the last response in the session
            self.session["last_payload"] = {"action": action, "payload": content}
            await sync_to_async(self.session_manager.update_billing_client_session)(
                self.scope["user"].id, self.session
            )
        else:
            await self.send_json(
                {
                    "action": "error",
                    "payload": {
                        "message": f"Invalid action: {action}",
                    },
                }
            )
            await self.close()

    async def get_started(self, content: dict[str, Any]) -> None:
        await self.update_session_action_and_payload(
            action="get_started", payload=content
        )
        await self.send_and_update_session_response(
            payload={
                "action": "get_started",
                "payload": {
                    "message": "Blast off! Please wait while we load your orders ready to be billed.",
                },
            }
        )

        await sleep(2)  # simulate loading time for testing
        await self.send_orders_ready()

    async def send_orders_ready(self) -> None:
        await self.update_session_action(action="orders_ready")
        await self.send_and_update_session_response(
            payload={
                "action": "orders_ready",
                "step": 1,
                "payload": {
                    "message": "Transferring user to orders ready to be billed.",
                },
            }
        )

    async def send_to_billing_queue(self, content: dict[str, Any]) -> None:
        await self.update_session_action_and_payload(
            action="send_to_billing_queue", payload=content
        )
        await self.send_and_update_session_response(
            payload={
                "action": "orders_ready",
                "step": 2,
                "payload": {
                    "message": "Please wait while we transfer your orders to the billing queue.",
                },
            }
        )

        billing_queue_response = await database_sync_to_async(
            transfer_to_billing_queue_service
        )(
            user_id=self.scope["user"].id,
            order_pros=content["payload"],
            task_id=str(uuid.uuid4()),
        )
        await sleep(2)  # simulate loading time for testing
        await self.send_and_update_session_response(
            payload={
                "action": "billing_queue",
                "step": 3,
                "payload": {
                    "message": billing_queue_response,
                },
            }
        )

    async def bill_orders(self, content: dict[str, Any]) -> None:
        await self.update_session_action_and_payload(
            action="bill_orders", payload=content
        )
        await self.send_and_update_session_response(
            payload={
                "action": "bill_orders",
                "step": 4,
                "payload": {
                    "status": "processing",
                    "message": "Please wait while we bill your orders.",
                },
            }
        )

        invoices = await database_sync_to_async(get_invoices_by_invoice_number)(
            invoices=content["payload"]
        )

        order_missing_info, billed_invoices = await database_sync_to_async(bill_orders)(
            user_id=self.scope["user"].id,
            invoices=invoices,
        )
        await sleep(2)  # simulate loading time for testing

        if len(order_missing_info) > 0:
            # Send failure message back to user and update session
            await self.send_and_update_session_response(
                payload={
                    "action": "bill_orders",
                    "step": 4,
                    "payload": {
                        "status": "failure",
                        "message": order_missing_info,
                    },
                }
            )
        if len(billed_invoices) > 0:
            # Send success message back to user and update session
            await self.send_and_update_session_response(
                payload={
                    "action": "bill_orders",
                    "step": 4,
                    "payload": {
                        "status": "success",
                        "message": f"Successfully billed {len(billed_invoices)} invoices",
                    },
                }
            )
            await sleep(2)  # simulate loading time for testing
            await self.good_job()

    async def good_job(self) -> None:
        # Send success message back to user and update session
        await self.send_and_update_session_response(
            payload={
                "action": "good_job",
                "step": 5,
                "payload": {
                    "message": "Good job! You have successfully billed your orders.",
                },
            }
        )

    async def update_session_action(self, *, action: str) -> None:
        self.session["previous_action"] = self.session["current_action"]
        self.session["current_action"] = action
        await sync_to_async(self.session_manager.update_billing_client_session)(
            self.scope["user"].id, self.session
        )

    async def update_session_action_and_payload(
        self, *, action: str, payload: dict[str, Any]
    ) -> None:
        self.session["previous_action"] = self.session["current_action"]
        self.session["last_payload"] = payload
        self.session["current_action"] = action
        await sync_to_async(self.session_manager.update_billing_client_session)(
            self.scope["user"].id, self.session
        )

    async def update_session_response(self, *, payload: dict[str, Any]) -> None:
        self.session["last_response"] = payload
        await sync_to_async(self.session_manager.update_billing_client_session)(
            self.scope["user"].id, self.session
        )

    async def send_and_update_session_response(
        self, *, payload: dict[str, Any]
    ) -> None:
        await self.send_json(payload)
        await self.update_session_response(payload=payload)
