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

import * as Yup from "yup";
import { ObjectSchema } from "yup";
import {
  AccessorialChargeFormValues,
  BillingControlFormValues,
  ChargeTypeFormValues,
} from "@/types/billing";
import {
  AutoBillingCriteriaChoicesProps,
  FuelMethodChoicesProps,
  OrderTransferCriteriaChoicesProps,
} from "@/utils/apps/billing";
import { StatusChoiceProps } from "@/types";
import { validateDecimal } from "@/lib/utils";

export const accessorialChargeSchema: ObjectSchema<AccessorialChargeFormValues> =
  Yup.object().shape({
    status: Yup.string<StatusChoiceProps>().required("Status is required"),
    code: Yup.string()
      .max(10, "Code must be less than 10 characters.")
      .required("Code is required"),
    description: Yup.string().notRequired(),
    isDetention: Yup.boolean().required("Detention is required"),
    chargeAmount: Yup.string()
      .test(
        "is-decimal",
        "Charge Amount cannot be more than four decimal places",
        (value) => {
          if (value !== "" && value !== undefined) {
            return validateDecimal(value, 4);
          }
          return true;
        },
      )
      .required(),
    method: Yup.string<FuelMethodChoicesProps>().required("Method is required"),
  });

export const chargeTypeSchema: ObjectSchema<ChargeTypeFormValues> =
  Yup.object().shape({
    status: Yup.string<StatusChoiceProps>().required("Status is required"),
    name: Yup.string()
      .max(50, "Name must be less than 50 characters.")
      .required("Name is required"),
    description: Yup.string()
      .max(100, "Description must be less than 100 characters.")
      .notRequired(),
  });

export const billingControlSchema: ObjectSchema<BillingControlFormValues> =
  Yup.object().shape({
    removeBillingHistory: Yup.boolean().required(
      "Remove billing history is required",
    ),
    autoBillOrders: Yup.boolean().required("Auto bill orders is required"),
    autoMarkReadyToBill: Yup.boolean().required(
      "Auto mark ready to bill is required",
    ),
    validateCustomerRates: Yup.boolean().required(
      "Validate customer rates is required",
    ),
    autoBillCriteria: Yup.string<AutoBillingCriteriaChoicesProps>().required(
      "Auto bill criteria is required",
    ),
    orderTransferCriteria:
      Yup.string<OrderTransferCriteriaChoicesProps>().required(
        "Order transfer criteria is required",
      ),
    enforceCustomerBilling: Yup.boolean().required(
      "Enforce customer billing is required",
    ),
  });
