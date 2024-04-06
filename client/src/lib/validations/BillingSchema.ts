/*
 * COPYRIGHT(c) 2024 Trenova
 *
 * This file is part of Trenova.
 *
 * The Trenova software is licensed under the Business Source License 1.1. You are granted the right
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

import { StatusChoiceProps } from "@/types";
import {
  AccessorialChargeFormValues,
  BillingControlFormValues,
  ChargeTypeFormValues,
  DocumentClassificationFormValues,
} from "@/types/billing";
import {
  AutoBillingCriteriaChoicesProps,
  FuelMethodChoicesProps,
  ShipmentTransferCriteriaChoicesProps,
} from "@/utils/apps/billing";
import * as Yup from "yup";
import { ObjectSchema } from "yup";

export const accessorialChargeSchema: ObjectSchema<AccessorialChargeFormValues> =
  Yup.object().shape({
    status: Yup.string<StatusChoiceProps>().required("Status is required"),
    code: Yup.string()
      .max(4, "Code must be less than 4 characters.")
      .required("Code is required"),
    description: Yup.string(),
    isDetention: Yup.boolean().required("Detention is required"),
    amount: Yup.number().required(),
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

export const documentClassSchema: ObjectSchema<DocumentClassificationFormValues> =
  Yup.object().shape({
    status: Yup.string<StatusChoiceProps>().required("Status is required"),
    code: Yup.string()
      .max(10, "Code must be less than 10 characters.")
      .required("Code is required"),
    description: Yup.string().max(
      100,
      "Description must be less than 100 characters.",
    ),
    color: Yup.string().max(100, "Color cannot be more than 100 characters"),
  });

export const billingControlSchema: ObjectSchema<BillingControlFormValues> =
  Yup.object().shape({
    removeBillingHistory: Yup.boolean().required(
      "Remove billing history is required",
    ),
    autoBillShipment: Yup.boolean().required("Auto bill shipment is required"),
    autoMarkReadyToBill: Yup.boolean().required(
      "Auto mark ready to bill is required",
    ),
    validateCustomerRates: Yup.boolean().required(
      "Validate customer rates is required",
    ),
    autoBillCriteria: Yup.string<AutoBillingCriteriaChoicesProps>().required(
      "Auto bill criteria is required",
    ),
    shipmentTransferCriteria:
      Yup.string<ShipmentTransferCriteriaChoicesProps>().required(
        "Order transfer criteria is required",
      ),
    enforceCustomerBilling: Yup.boolean().required(
      "Enforce customer billing is required",
    ),
  });
