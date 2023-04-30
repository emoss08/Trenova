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
from django.db.models.signals import post_save, pre_save, pre_delete


class OrganizationConfig(AppConfig):
    default_auto_field = "django.db.models.BigAutoField"
    name = "organization"

    def ready(self) -> None:
        from organization import signals

        # Organization
        post_save.connect(
            signals.create_dispatch_control,
            sender="organization.Organization",
            dispatch_uid="create_dispatch_control",
        )
        post_save.connect(
            signals.create_order_control,
            sender="organization.Organization",
            dispatch_uid="create_order_control",
        )
        post_save.connect(
            signals.create_route_control,
            sender="organization.Organization",
            dispatch_uid="create_route_control",
        )
        post_save.connect(
            signals.create_billing_control,
            sender="organization.Organization",
            dispatch_uid="create_billing_control",
        )
        post_save.connect(
            signals.create_email_control,
            sender="organization.Organization",
            dispatch_uid="create_email_control",
        )
        post_save.connect(
            signals.create_kube_configuration,
            sender="organization.Organization",
            dispatch_uid="create_kube_configuration",
        )
        post_save.connect(
            signals.create_invoice_control,
            sender="organization.Organization",
            dispatch_uid="create_invoice_control",
        )
        post_save.connect(
            signals.create_depot_detail,
            sender="organization.Depot",
            dispatch_uid="create_depot_detail",
        )

        # Table Change Alerts
        post_save.connect(
            signals.create_trigger_signal,
            sender="organization.TableChangeAlert",
            dispatch_uid="create_trigger_signal",
        )
        pre_save.connect(
            signals.save_trigger_name_requirements,
            sender="organization.TableChangeAlert",
            dispatch_uid="save_trigger_name_requirements",
        )
        pre_save.connect(
            signals.delete_and_recreate_trigger_and_function,
            sender="organization.TableChangeAlert",
            dispatch_uid="delete_and_recreate_trigger_and_function",
        )
        pre_delete.connect(
            signals.drop_trigger_and_function_signal,
            sender="organization.TableChangeAlert",
            dispatch_uid="drop_trigger_and_function_signal",
        )
        pre_delete.connect(
            signals.delete_and_add_new_trigger,
            sender="organization.TableChangeAlert",
            dispatch_uid="delete_and_add_new_trigger",
        )
