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
from __future__ import annotations

import uuid
from enum import Enum
from typing import Any, TypedDict, Union
from uuid import UUID

from django.db.models import UUIDField
from django.http import HttpRequest
from rest_framework.request import Request

from accounts.models import User

type ModelUUID = Union[UUIDField[Union[str, UUID, None], UUID], Any]
type HealthStatus = Union[dict[str, Union[str, int, int, int]]]
type HealthStatusAndTime = Union[dict[str, Union[str, int, int, int, float, float]]]
type DiskUsage = tuple[int, int, int]
type BilledShipments = tuple[list[Any | list[dict[str, str | list[str]]]], list[Any]]
type EDIEnvelope = tuple[str, str, str, str, str, str]
type ModelDelete = tuple[int, dict[str, int]]
type Coordinates = tuple[
    tuple[float | None, float | None], tuple[float | None, float | None]
] | None


class BillingClientActions(Enum):
    """
    The different actions that the billing client can take.
    """

    GET_STARTED = "GET_STARTED"
    SHIPMENTS_READY = "SHIPMENTS_READY"
    BILLING_QUEUE = "BILLING_QUEUE"
    BILL_shipmentS = "BILL_shipmentS"
    BILLING_COMPLETE = "BILLING_COMPLETE"


class BillingClientStatuses(Enum):
    """
    The different statuses that the billing client can have.
    """

    SUCCESS = "SUCCESS"
    PROCESSING = "PROCESSING"
    FAILURE = "FAILURE"
    WARNING = "WARNING"
    INFO = "INFO"


class BillingClientSessionResponse(TypedDict):
    """
    A response from the billing client session.
    """

    user_id: uuid.UUID
    # The current action that the client is taking.
    current_action: str
    # The previous action that was taken by the client.
    previous_action: str | None
    # Last response sent from the server to the client.
    last_response: Any
    # Last message sent from the client to the server.
    last_message: Any


class AuthenticatedRequest[H: Request]:
    """
    A request that has been authenticated by the authentication middleware.

    Attributes:
        user: The user that made the request.
    """

    user: User


class AuthenticatedHttpRequest[T: HttpRequest]:
    user: User


class BillingClientResponse(TypedDict):
    """
    A response from the billing client.
    """

    action: str
    status: BillingClientStatuses
    step: int
    message: Any
