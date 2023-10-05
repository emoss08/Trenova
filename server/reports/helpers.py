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

# Allowed models for reports
ALLOWED_MODELS = {
    "User": {
        "app_label": "accounts",
        "allowed_fields": [
            {"value": "username", "label": "Username"},
            {"value": "email", "label": "Email"},
            {"value": "date_joined", "label": "Date Joined"},
            {"value": "is_staff", "label": "Is Staff"},
            {"value": "profiles__first_name", "label": "First Name"},
            {"value": "profiles__last_name", "label": "Last Name"},
            {"value": "profiles__address_line_1", "label": "Address Line 1"},
            {"value": "profiles__address_line_2", "label": "Address Line 2"},
            {"value": "profiles__city", "label": "City"},
            {"value": "profiles__state", "label": "State"},
            {"value": "profiles__zip_code", "label": "Zip Code"},
            {"value": "profiles__phone_number", "label": "Phone Number"},
            {
                "value": "profiles__is_phone_verified",
                "label": "Is Phone Verified",
            },
            {"value": "profiles__job_title__name", "label": "Job Title Name"},
            {
                "value": "profiles__job_title__description",
                "label": "Job Title Description",
            },
            {"value": "department__name", "label": "Department Name"},
            {
                "value": "department__description",
                "label": "Department Description",
            },
            {"value": "organization__name", "label": "Organization Name"},
        ],
    },
    "UserProfile": {
        "app_label": "accounts",
        "allowed_fields": [
            {"value": "user__username", "label": "Username"},
            {"value": "user__email", "label": "Email"},
            {"value": "user__date_joined", "label": "Date Joined"},
            {"value": "user__is_staff", "label": "Is Staff"},
            {"value": "first_name", "label": "First Name"},
            {"value": "last_name", "label": "Last Name"},
            {"value": "address_line_1", "label": "Address Line 1"},
            {"value": "address_line_2", "label": "Address Line 2"},
            {"value": "city", "label": "City"},
            {"value": "state", "label": "State"},
            {"value": "zip_code", "label": "Zip Code"},
            {"value": "phone_number", "label": "Phone Number"},
            {
                "value": "is_phone_verified",
                "label": "Is Phone Verified",
            },
        ],
    },
    "JobTitle": {
        "app_label": "accounts",
        "allowed_fields": [
            {"value": "organization__name", "label": "Organization Name"},
            {"value": "status", "label": "Status"},
            {"value": "name", "label": "Name"},
            {"value": "description", "label": "Description"},
            {"value": "job_function", "label": "Job Function"},
        ],
    },
    "Organization": {
        "app_label": "organization",
        "allowed_fields": [
            {"value": "name", "label": "Name"},
            {"value": "scac_code", "label": "SCAC Code"},
            {"value": "dot_number", "label": "DOT Number"},
            {"value": "address_line_1", "label": "Address Line 1"},
            {"value": "address_line_2", "label": "Address Line 2"},
            {"value": "city", "label": "City"},
            {"value": "state", "label": "State"},
            {"value": "zip_code", "label": "Zip Code"},
            {"value": "phone_number", "label": "Phone Number"},
            {"value": "website", "label": "Website"},
            {"value": "org_type", "label": "Organization Type"},
            {"value": "timezone", "label": "Timezone"},
            {"value": "language", "label": "Language"},
            {"value": "currency", "label": "Currency"},
            {"value": "date_format", "label": "Date Format"},
            {"value": "time_format", "label": "Time Format"},
            {"value": "token_expiration_days", "label": "Token Expiration Days"},
        ],
    },
    "Department": {
        "app_label": "organization",
        "allowed_fields": [
            {"value": "organization__name", "label": "Organization Name"},
            {"value": "name", "label": "Name"},
            {"value": "description", "label": "Description"},
        ],
    },
    "DivisionCode": {
        "app_label": "accounting",
        "allowed_fields": [
            {"value": "organization__name", "label": "Organization Name"},
            {"value": "status", "label": "Status"},
            {"value": "code", "label": "Code"},
            {"value": "description", "label": "Description"},
            {"value": "cash_account__account_number", "label": "Cash Account"},
            {"value": "ap_account__account_number", "label": "AP Account"},
            {"value": "expense_account__account_number", "label": "Expense Account"},
        ],
    },
    "RevenueCode": {
        "app_label": "accounting",
        "allowed_fields": [
            {"value": "organization__name", "label": "Organization Name"},
            {"value": "code", "label": "Code"},
            {"value": "description", "label": "Description"},
            {"value": "expense_account__account_number", "label": "Expense Account"},
            {"value": "revenue_account__account_number", "label": "Revenue Account"},
        ],
    },
    "GeneralLedgerAccount": {
        "app_label": "accounting",
        "allowed_fields": [
            {"value": "organization__name", "label": "Organization Name"},
            {"value": "status", "label": "Status"},
            {"value": "account_number", "label": "Account Number"},
            {"value": "description", "label": "Description"},
            {"value": "account_type", "label": "Account Type"},
            {"value": "cash_flow_type", "label": "Cash Flow Type"},
            {"value": "account_sub_type", "label": "Account Sub Type"},
            {"value": "account_classification", "label": "Account Classification"},
        ],
    },
    "ChargeType": {
        "app_label": "billing",
        "allowed_fields": [
            {"value": "organization__name", "label": "Organization Name"},
            {"value": "name", "label": "Name"},
            {"value": "description", "label": "Description"},
        ],
    },
    "AccessorialCharge": {
        "app_label": "billing",
        "allowed_fields": [
            {"value": "organization__name", "label": "Organization Name"},
            {"value": "code", "label": "Code"},
            {"value": "description", "label": "Description"},
            {"value": "is_detention", "label": "Is Detention"},
            {"value": "charge_amount", "label": "Charge Amount"},
            {"value": "method", "label": "Method"},
        ],
    },
    "HazardousMaterial": {
        "app_label": "commodities",
        "allowed_fields": [
            {"value": "organization__name", "label": "Organization Name"},
            {"value": "status", "label": "Status"},
            {"value": "name", "label": "Name"},
            {"value": "description", "label": "Description"},
            {"value": "hazard_class", "label": "Hazard Class"},
            {"value": "packing_group", "label": "Packing Group"},
            {"value": "erg_number", "label": "ERG Number"},
            {"value": "proper_shipping_name", "label": "Proper Shipping Name"},
        ],
    },
    "Commodity": {
        "app_label": "commodities",
        "allowed_fields": [
            {"value": "organization__name", "label": "Organization Name"},
            {"value": "name", "label": "Name"},
            {"value": "description", "label": "Description"},
            {"value": "min_temp", "label": "Minimum Temperature"},
            {"value": "max_temp", "label": "Maximum Temperature"},
            {"value": "set_point_temp", "label": "Set Point Temperature"},
            {"value": "unit_of_measure", "label": "Unit of Measure"},
            {"value": "hazmat__status", "label": "Hazardous Material Status"},
            {"value": "hazmat__name", "label": "Hazardous Material Name"},
            {"value": "hazmat__description", "label": "Hazardous Material Description"},
            {
                "value": "hazmat__hazard_class",
                "label": "Hazardous Material Hazard Class",
            },
            {
                "value": "hazmat__packing_group",
                "label": "Hazardous Material Packing Group",
            },
            {"value": "hazmat__erg_number", "label": "Hazardous Material ERG Number"},
            {
                "value": "hazmat__proper_shipping_name",
                "label": "Hazardous Material Proper Shipping Name",
            },
            {"value": "is_hazmat", "label": "Is Hazardous Material"},
        ],
    },
    "Customer": {
        "app_label": "customer",
        "allowed_fields": [
            {"value": "organization__name", "label": "Organization Name"},
            {"value": "status", "label": "Status"},
            {"value": "code", "label": "Code"},
            {"value": "name", "label": "Name"},
            {"value": "address_line_1", "label": "Address Line 1"},
            {"value": "address_line_2", "label": "Address Line 2"},
            {"value": "city", "label": "City"},
            {"value": "zip_code", "label": "Zip Code"},
            {"value": "has_customer_portal", "label": "Has Customer Portal"},
            {"value": "auto_mark_ready_to_bill", "label": "Auto Mark Ready To Bill"},
        ],
    },
    "DelayCode": {
        "app_label": "dispatch",
        "allowed_fields": [
            {"value": "organization__name", "label": "Organization Name"},
            {"value": "code", "label": "Code"},
            {"value": "description", "label": "Description"},
            {"value": "f_carrier_or_driver", "label": "F Carrier Or Driver"},
        ],
    },
    "FleetCode": {
        "app_label": "dispatch",
        "allowed_fields": [
            {"value": "organization__name", "label": "Organization Name"},
            {"value": "is_active", "label": "Is Active"},
            {"value": "code", "label": "Code"},
            {"value": "description", "label": "Description"},
            {"value": "revenue_goal", "label": "Revenue Goal"},
            {"value": "deadhead_goal", "label": "Deadhead Goal"},
            {"value": "mileage_goal", "label": "Mileage Goal"},
            {"value": "manager__username", "label": "Manager Username"},
        ],
    },
    "CommentType": {
        "app_label": "dispatch",
        "allowed_fields": [
            {"value": "organization__name", "label": "Organization Name"},
            {"value": "name", "label": "Name"},
            {"value": "description", "label": "Description"},
            {"value": "created", "label": "Created At"},
            {"value": "modified", "label": "Modified At"},
        ],
    },
    "Rate": {
        "app_label": "dispatch",
        "allowed_fields": [
            {"value": "organization__name", "label": "Organization Name"},
            {"value": "rate_number", "label": "Rate Number"},
            {"value": "customer__name", "label": "Customer Name"},
            {"value": "customer__code", "label": "Customer Code"},
            {"value": "effective_date", "label": "Effective Date"},
            {"value": "expiration_date", "label": "Expiration Date"},
            {"value": "commodity__name", "label": "Commodity Name"},
            {"value": "commodity__description", "label": "Commodity Description"},
            {"value": "shipment_type__name", "label": "shipment type Name"},
            {"value": "equipment_type__name", "label": "Equipment Type Name"},
            {"value": "origin_location__code", "label": "Origin Location Code"},
            {
                "value": "destination_location__code",
                "label": "Destination Location Code",
            },
            {"value": "rate_method", "label": "Rate Method"},
            {"value": "rate_amount", "label": "Rate Amount"},
            {"value": "distance_override", "label": "Distance Override"},
            {"value": "comments", "label": "Comments"},
        ],
    },
    "EquipmentType": {
        "app_label": "equipment",
        "allowed_fields": [
            {"value": "name", "label": "Name"},
            {"value": "description", "label": "Description"},
            {"value": "cost_per_mile", "label": "Cost Per Mile"},
        ],
    },
    "EquipmentManufacturer": {
        "app_label": "equipment",
        "allowed_fields": [
            {"value": "organization__name", "label": "Organization Name"},
            {"value": "status", "label": "Status"},
            {"value": "name", "label": "Name"},
            {"value": "description", "label": "Description"},
            {"value": "created", "label": "Created"},
            {"value": "modified", "label": "Modified"},
        ],
    },
}
