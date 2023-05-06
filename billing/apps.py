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
from django.db.models.signals import post_delete, post_save, pre_delete, pre_save

from core.signals import invalidate_cache


class BillingConfig(AppConfig):
    default_auto_field = "django.db.models.BigAutoField"
    name = "billing"

    def ready(self) -> None:
        from billing import signals

        # Document Classification
        pre_delete.connect(
            signals.prevent_delete_on_rate_con_doc_class,
            sender="billing.DocumentClassification",
            dispatch_uid="prevent_delete_on_rate_con_doc_class",
        )

        # Billing Queue
        pre_save.connect(
            signals.generate_invoice_number_on_billing_queue,
            sender="billing.BillingQueue",
            dispatch_uid="generate_invoice_number_billing_queue",
        )
        pre_save.connect(
            signals.transfer_order_details_to_billing_queue,
            sender="billing.BillingQueue",
            dispatch_uid="transfer_order_details_billing_queue",
        )

        # Billing History
        pre_save.connect(
            signals.save_invoice_number_on_billing_history,
            sender="billing.BillingHistory",
            dispatch_uid="save_invoice_number_billing_history",
        )
        pre_save.connect(
            signals.transfer_order_details_to_billing_history,
            sender="billing.BillingHistory",
            dispatch_uid="transfer_order_details_billing_history",
        )
        pre_delete.connect(
            signals.check_billing_history,
            sender="billing.BillingHistory",
            dispatch_uid="check_billing_history",
        )

        # Billing Control Cache Invalidations
        post_save.connect(invalidate_cache, sender="billing.BillingControl")
        post_delete.connect(invalidate_cache, sender="billing.BillingControl")
