/*
 * COPYRIGHT(c) 2023 MONTA
 *
 * This file is part of Monta.
 *
 * The Monta software is licensed under the Business Source License 1.1. You are granted the right
 * to copy, modify, and redistribute the software, but only for non-production use or with a total
 * of less than three server instances. Starting from the Change Date (November 16, 2026), the
 * software will be made available under version 2 or later of the GNU General Public License.
 * If you use the software in violation of this license, your rights under the license will be
 * terminated automatically. The software is provided "as is," and the Licensor disclaims all
 * warranties and conditions. If you use this license's text or the "Business Source License" name
 * and trademark, you must comply with the Licensor's covenants, which include specifying the
 * Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
 * Grant, and not modifying the license in any other way.
 */

import { ObjectSchema } from "yup";
import {
  AccessorialChargeFormValues,
  BillingControlFormValues,
  ChargeTypeFormValues,
} from "@/types/apps/billing";
import * as Yup from "yup";
import {
  AutoBillingCriteriaChoicesProps,
  fuelMethodChoicesProps,
  OrderTransferCriteriaChoicesProps,
} from "@/utils/apps/billing";

export const accessorialChargeSchema: ObjectSchema<AccessorialChargeFormValues> =
  Yup.object().shape({
    code: Yup.string().required("Code is required"),
    description: Yup.string().required("Description is required"),
    is_detention: Yup.boolean().required("Detention is required"),
    charge_amount: Yup.number().required("Charge amount is required"),
    method: Yup.string<fuelMethodChoicesProps>().required("Method is required"),
  });

export const chargeTypeSchema: ObjectSchema<ChargeTypeFormValues> =
  Yup.object().shape({
    name: Yup.string().required("Name is required"),
    description: Yup.string().notRequired(),
  });

export const billingControlSchema: ObjectSchema<BillingControlFormValues> =
  Yup.object().shape({
    remove_billing_history: Yup.boolean().required(
      "Remove billing history is required"
    ),
    auto_bill_orders: Yup.boolean().required("Auto bill orders is required"),
    auto_mark_ready_to_bill: Yup.boolean().required(
      "Auto mark ready to bill is required"
    ),
    validate_customer_rates: Yup.boolean().required(
      "Validate customer rates is required"
    ),
    auto_bill_criteria: Yup.string<AutoBillingCriteriaChoicesProps>().required(
      "Auto bill criteria is required"
    ),
    order_transfer_criteria:
      Yup.string<OrderTransferCriteriaChoicesProps>().required(
        "Order transfer criteria is required"
      ),
    enforce_customer_billing: Yup.boolean().required(
      "Enforce customer billing is required"
    ),
  });
