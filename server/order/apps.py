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

from django.apps import AppConfig
from django.db.models.signals import post_save, pre_delete, pre_save


class OrderConfig(AppConfig):
    default_auto_field = "django.db.models.BigAutoField"
    name = "order"

    def ready(self) -> None:
        from order import signals

        pre_save.connect(
            signals.set_order_pro_number,
            sender="order.Order",
            dispatch_uid="set_order_pro_number",
        )
        post_save.connect(
            signals.create_order_initial_movement,
            sender="order.Order",
            dispatch_uid="create_order_initial_movement",
        )
        pre_delete.connect(
            signals.check_order_removal_policy,
            sender="order.Order",
            dispatch_uid="check_order_removal_policy",
        )
        post_save.connect(
            signals.set_order_mileage_and_create_route,
            sender="order.Order",
            dispatch_uid="set_order_mileage_and_create_route",
        )
