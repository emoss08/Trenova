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
from typing import Any

from cacheops import invalidate_obj
from django.db import models

logger = logging.getLogger(__name__)


def invalidate_cache(
    sender: type[models.Model], instance: type[models.Model], **kwargs: Any
) -> None:
    """The invalidate_cache function is a signal receiver that invalidates the cache for an object.

    Args:
        sender: type[models.Model]: Specify the type of object that will be sent to this function
        instance: type[models.Model]: Specify the type of object that is passed to the function
        **kwargs: Any: Catch any extra parameters that are passed to the function

    Returns:
        None: This function does not return anything.
    """
    logger.debug(f"Invalidating cache for {instance}")
    invalidate_obj(instance)
