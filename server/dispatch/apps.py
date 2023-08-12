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

from core.signals import invalidate_cache
from django.apps import AppConfig
from django.db.models.signals import post_delete, post_save, pre_save


class DispatchConfig(AppConfig):
    default_auto_field = "django.db.models.BigAutoField"
    name = "dispatch"

    def ready(self) -> None:
        from dispatch import signals

        pre_save.connect(
            signals.set_rate_number,
            sender="dispatch.Rate",
            dispatch_uid="set_rate_number",
        )
        pre_save.connect(
            signals.set_charge_amount_on_billing_table,
            sender="dispatch.RateBillingTable",
            dispatch_uid="set_charge_amount_on_billing_table",
        )

        # Dispatch Control cache invalidations
        post_save.connect(invalidate_cache, sender="dispatch.DispatchControl")
        post_delete.connect(invalidate_cache, sender="dispatch.DispatchControl")
