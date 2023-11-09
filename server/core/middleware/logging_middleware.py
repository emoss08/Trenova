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

import logging

from django.utils.deprecation import MiddlewareMixin
from django.utils.timezone import now

logger = logging.getLogger(__name__)


class RocketLoggingMiddleware(MiddlewareMixin):
    def process_response(self, request, response):
        duration = (now() - request._logging_start_time).total_seconds()
        handler = (
            request.resolver_match.url_name if request.resolver_match else "UNKNOWN"
        )
        client = f"{request.META.get('REMOTE_ADDR', '')}:{request.META.get('REMOTE_PORT', 'UNKNOWN')}"

        logger = logging.getLogger("django")
        logger.info(
            "HTTP Request",
            extra={
                "method": request.method,
                "path": request.get_full_path(),
                "status": response.status_code,
                "handler": handler,
                "time_taken": duration,
                "client": client,
                "size": len(response.content),
            },
        )

        return response

    def process_request(self, request):
        request._logging_start_time = now()
