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

from typing import Any

from django.db.models.signals import post_save
from django.dispatch import receiver

from stops import models
from stops.services import generation


@receiver(post_save, sender=models.Stop)
def sequence_stops(
    sender: models.Stop, instance: models.Stop, created: bool, **kwargs: Any
) -> None:
    """Sequence Stops
    Sequence the stops when a new stop is added
    to a movement.
    Args:
        sender (Stop): Stop
        instance (Stop): The stop instance.
        created (bool): if the Stop was created.
        **kwargs (Any): Keyword arguments.
    Returns:
        None
    """
    if created:
        generation.StopService.sequence_stops(instance)
