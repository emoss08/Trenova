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
from billing import consumers
from django.urls import re_path
from organization.consumers import KeepAliveConsumer, LogoutConsumer
from reports.consumers import NotificationConsumer

websocket_urlpatterns = [
    re_path(r"ws/keepalive/$", KeepAliveConsumer.as_asgi()),
    re_path(r"ws/notifications/$", NotificationConsumer.as_asgi()),
    re_path(r"ws/billing_client/$", consumers.BillingClientConsumer.as_asgi()),  # type: ignore
    re_path(r"ws/session/(?P<user_id>[0-9a-f-]+)/$", LogoutConsumer.as_asgi()),
]
