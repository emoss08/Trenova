/*
 * COPYRIGHT(c) 2024 MONTA
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
import { validateDecimal } from "@/lib/utils";
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
  CommentTypeFormValues,
  DelayCodeFormValues,
  DispatchControlFormValues,
  FeasibilityToolControlFormValues,
  FleetCodeFormValues,
  RateBillingTableFormValues,
  RateFormValues,
} from "@/types/dispatch";
import { TRateMethodChoices } from "@/lib/constants";
import { StatusChoiceProps } from "@/types";
import {
  FeasibilityOperatorChoiceProps,
  ServiceIncidentControlChoiceProps,
} from "@/lib/choices";

export const dispatchControlSchema: ObjectSchema<DispatchControlFormValues> =
  Yup.object().shape({
    recordServiceIncident:
      Yup.string<ServiceIncidentControlChoiceProps>().required(
        "Record Service Incident is required",
      ),
    gracePeriod: Yup.number().required("Grace Period is required"),
    deadheadTarget: Yup.number().required("Deadhead Target is required"),
    enforceWorkerAssign: Yup.boolean().required("Worker Assign is required"),
    trailerContinuity: Yup.boolean().required("Trailer Continuity is required"),
    dupeTrailerCheck: Yup.boolean().required("Dupe Trailer Check is required"),
    maintenanceCompliance: Yup.boolean().required(
      "Maintenance Compliance is required",
    ),
    maxShipmentWeightLimit: Yup.number().required(
      "Max Load Weight Limit is required",
    ),
    regulatoryCheck: Yup.boolean().required("Regulatory Check is required"),
    prevShipmentsOnHold: Yup.boolean().required(
      "Previous Shipments on Hold is required",
    ),
    workerTimeAwayRestriction: Yup.boolean().required(
      "Worker Time Away Restriction is required",
    ),
    tractorWorkerFleetConstraint: Yup.boolean().required(
      "Tractor Worker Fleet Constraint is required",
    ),
  });

export const feasibilityControlSchema: ObjectSchema<FeasibilityToolControlFormValues> =
  Yup.object().shape({
    mpwOperator: Yup.string<FeasibilityOperatorChoiceProps>().required(
      "Miles per week operator is required.",
    ),
    mpwCriteria: Yup.number().required("Miles per week criteria is required."),
    mpdOperator: Yup.string<FeasibilityOperatorChoiceProps>().required(
      "Miles per day operator is required.",
    ),
    mpdCriteria: Yup.number().required("Miles per day criteria is required."),
    mpgOperator: Yup.string<FeasibilityOperatorChoiceProps>().required(
      "Miles per gallon operator is required.",
    ),
    mpgCriteria: Yup.number().required(
      "Miles per gallon criteria is required.",
    ),
    otpOperator: Yup.string<FeasibilityOperatorChoiceProps>().required(
      "On-time performance operator is required.",
    ),
    otpCriteria: Yup.number().required(
      "On-time performance criteria is required.",
    ),
  });

export const delayCodeSchema: ObjectSchema<DelayCodeFormValues> =
  Yup.object().shape({
    status: Yup.string<StatusChoiceProps>().required("Status is required"),
    code: Yup.string()
      .required("Name is required")
      .max(4, "Code cannot be more than 4 characters"),
    description: Yup.string()
      .required("Description is required")
      .max(100, "Description cannot be more than 100 characters"),
    fCarrierOrDriver: Yup.boolean().required("Carrier or Driver is required"),
  });

export const fleetCodeSchema: ObjectSchema<FleetCodeFormValues> =
  Yup.object().shape({
    status: Yup.string<StatusChoiceProps>().required("Status is required"),
    code: Yup.string()
      .required("Name is required")
      .max(10, "Code cannot be more than 10 characters"),
    revenueGoal: Yup.string()
      .notRequired()
      .nullable()
      .test(
        "is-decimal",
        "Revenue Goal must be a decimal with no more than two decimal places",
        (value) => {
          if (value === undefined || value === null || value === "") {
            return true; // Passes validation for null, undefined, or empty string
          }
          return validateDecimal(value, 2);
        },
      ),
    deadheadGoal: Yup.string()
      .notRequired()
      .nullable()
      .test(
        "is-decimal",
        "Deadhead Goal must be a decimal with no more than two decimal places",
        (value) => {
          if (value === undefined || value === null || value === "") {
            return true; // Passes validation for null, undefined, or empty string
          }
          return validateDecimal(value, 2);
        },
      ),
    mileageGoal: Yup.string()
      .notRequired()
      .nullable()
      .test(
        "is-decimal",
        "Mileage Goal must be a decimal with no more than two decimal places",
        (value) => {
          if (value === undefined || value === null || value === "") {
            return true; // Passes validation for null, undefined, or empty string
          }
          return validateDecimal(value, 2);
        },
      ),
    description: Yup.string()
      .required("Description is required")
      .max(100, "Description cannot be more than 100 characters"),
    manager: Yup.string().nullable().notRequired(),
  });

export const commentTypeSchema: ObjectSchema<CommentTypeFormValues> =
  Yup.object().shape({
    status: Yup.string<StatusChoiceProps>().required("Status is required"),
    name: Yup.string()
      .max(10, "Name cannot be more than 10 characters")
      .required("Name is required"),
    description: Yup.string()
      .max(100, "Description cannot be more than 100 characters")
      .required("Description is required"),
  });

export const rateBillingTableSchema: ObjectSchema<RateBillingTableFormValues> =
  Yup.object().shape({
    accessorialCharge: Yup.string().required("Accessorial Charge is required"),
    description: Yup.string()
      .max(100, "Description cannot be more than 100 characters long")
      .notRequired(),
    unit: Yup.number().required("Unit is required"),
    chargeAmount: Yup.number().required("Charge Amount is required"),
    subTotal: Yup.number().required("Subtotal is required"),
  });

export const rateSchema: ObjectSchema<RateFormValues> = Yup.object().shape({
  isActive: Yup.boolean().required("Name is required"),
  rateNumber: Yup.string()
    .max(6, "Rate Number cannot be more than 6 characters")
    .required("Rate Number is required"),
  customer: Yup.string().nullable().notRequired(),
  effectiveDate: Yup.date().required("Effective Date is required"),
  expirationDate: Yup.date().required("Expiration Date is required"),
  commodity: Yup.string().nullable().notRequired(),
  orderType: Yup.string().nullable().notRequired(),
  equipmentType: Yup.string().nullable().notRequired(),
  originLocation: Yup.string().nullable().notRequired(),
  destinationLocation: Yup.string().nullable().notRequired(),
  rateMethod: Yup.string<TRateMethodChoices>().required(
    "Rate Method is required",
  ),
  rateAmount: Yup.number().required("Rate Amount is required"),
  distanceOverride: Yup.number().nullable().notRequired(),
  comments: Yup.string().nullable().notRequired(),
  rateBillingTables: Yup.array()
    .of(rateBillingTableSchema)
    .notRequired()
    .nullable(),
});
