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
from typing import Any, TypeAlias, TypedDict, Union
from uuid import UUID

from django.db.models import UUIDField
from django.http import HttpRequest
from rest_framework.request import Request

from accounts.models import User

ModelUUID: TypeAlias = Union[UUIDField[Union[str, UUID, None], UUID], Any]
HealthStatus: TypeAlias = Union[dict[str, Union[str, int, int, int]]]
HealthStatusAndTime: TypeAlias = Union[
    dict[str, Union[str, int, int, int, float, float]]
]
DiskUsage: TypeAlias = tuple[int, int, int]
BilledShipments: TypeAlias = tuple[
    list[Any | list[dict[str, str | list[str]]]], list[Any]
]
EDIEnvelope: TypeAlias = tuple[str, str, str, str, str, str]
ModelDelete: TypeAlias = tuple[int, dict[str, int]]


class BillingClientActions(Enum):
    """
    The different actions that the billing client can take.
    """

    GET_STARTED = "GET_STARTED"
    shipments_READY = "shipments_READY"
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


class AuthenticatedRequest(Request):
    """
    A request that has been authenticated by the authentication middleware.

    Attributes:
        user: The user that made the request.
    """

    user: User


class AuthenticatedHttpRequest(HttpRequest):
    user: User


class BillingClientResponse(TypedDict):
    """
    A response from the billing client.
    """

    action: str
    status: BillingClientStatuses
    step: int
    message: Any
