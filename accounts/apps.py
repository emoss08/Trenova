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
from django.db.models.signals import post_delete, post_save


class AccountConfig(AppConfig):
    default_auto_field = "django.db.models.BigAutoField"
    name = "accounts"

    def ready(self) -> None:
        from accounts.signals import add_expiration_to_token
        from core import signals

        # Token Expiration Date
        post_save.connect(add_expiration_to_token, sender="accounts.Token")

        # JobTitle Cache Invalidations
        post_save.connect(
            signals.invalidate_cache,
            sender="accounts.JobTitle",
        )
        post_delete.connect(
            signals.invalidate_cache,
            sender="accounts.JobTitle",
        )

        # User Cache Invalidations
        post_save.connect(signals.invalidate_cache, sender="accounts.User")
        post_delete.connect(signals.invalidate_cache, sender="accounts.User")

        # User Profile Cache Invalidations
        post_save.connect(signals.invalidate_cache, sender="accounts.UserProfile")
        post_delete.connect(signals.invalidate_cache, sender="accounts.UserProfile")

        # Auth Group Cache Invalidations
        post_save.connect(
            signals.invalidate_cache,
            sender="auth.Group",
        )
        post_delete.connect(
            signals.invalidate_cache,
            sender="auth.Group",
        )

        # Auth Permission Cache Invalidations
        post_save.connect(
            signals.invalidate_cache,
            sender="auth.Permission",
        )
        post_delete.connect(
            signals.invalidate_cache,
            sender="auth.Permission",
        )
