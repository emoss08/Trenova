# --------------------------------------------------------------------------------------------------
#  COPYRIGHT(c) 2024 Trenova                                                                       -
#                                                                                                  -
#  This file is part of Trenova.                                                                   -
#                                                                                                  -
#  The Trenova software is licensed under the Business Source License 1.1. You are granted the right
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

from enum import Enum

KAFKA_EXCLUDE_TOPIC_PREFIXES = [
    "trenova_app_.public.silk_",
    "trenova_app_.public.auditlog_",
    "trenova_app_.public.admin_",
    "trenova_app_.public.django_",
    "trenova_app_.public.auth_",
    "trenova_app_.public.states",
    "trenova_app_.public.flag",
    "trenova_app_.public.user",
    "trenova_app_.public.a_group",
    "trenova_app_.public.audit_",
    "trenova_app_.public.user",
    "trenova_app_.public.organization",
    "trenova_app_.public.business_unit",
    "trenova_app_.public.plugin",
    "trenova_app_.public.waffle_",
    "trenova_app_.public.edi",
    "trenova_app_.public.states",
    "trenova_app_.public.document",
    "trenova_app_.public.accounting_control",
    "trenova_app_.public.billing_control",
    "trenova_app_.public.doc_template_customization",
    "trenova_app_.public.scheduled_report",
    "trenova_app_.public.weekday",
    "trenova_app_.public.notification_setting",
    "trenova_app_.public.notification_type",
    "trenova_app_.public.route_control",
    "trenova_app_.public.feasibility_tool_control",
    "trenova_app_.public.google_api",
    "trenova_app_.public.integration",
    "trenova_app_.public.shipment_control",
    "trenova_app_.public.formula_template",
    "trenova_app_.public.dispatch_control",
    "trenova_app_.public.email_control",
    "trenova_app_.public.invoice_control",
    "trenova_app_.public.tax_rate",
    "trenova_app_.public.template",
    "trenova_app_.public.custom_report",
    "trenova_app_.public.feature_flag",
    "my_connect_offsets",
    "my_connect_configs",
    "my_connect_statuses",
    "__",
]


class ActionChoices(str, Enum):
    INSERT = "INSERT"
    UPDATE = "UPDATE"
    DELETE = "DELETE"
    BOTH = "BOTH"  # INSERT and UPDATE
    ALL = "ALL"
