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
  DelayCodeFormValues,
  DispatchControlFormValues,
  FleetCodeFormValues,
} from "@/types/dispatch";

export const dispatchControlSchema: ObjectSchema<DispatchControlFormValues> =
  Yup.object().shape({
    recordServiceIncident: Yup.string().required(
      "Record Service Incident is required",
    ),
    gracePeriod: Yup.number().required("Grace Period is required"),
    deadheadTarget: Yup.number().required("Deadhead Target is required"),
    driverAssign: Yup.boolean().required("Driver Assign is required"),
    trailerContinuity: Yup.boolean().required("Trailer Continuity is required"),
    dupeTrailerCheck: Yup.boolean().required("Dupe Trailer Check is required"),
    regulatoryCheck: Yup.boolean().required("Regulatory Check is required"),
    prevOrdersOnHold: Yup.boolean().required(
      "Previous Orders on Hold is required",
    ),
    driverTimeAwayRestriction: Yup.boolean().required(
      "Driver Time Away Restriction is required",
    ),
    tractorWorkerFleetConstraint: Yup.boolean().required(
      "Tractor Worker Fleet Constraint is required",
    ),
  });

export const delayCodeSchema: ObjectSchema<DelayCodeFormValues> =
  Yup.object().shape({
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
    code: Yup.string()
      .required("Name is required")
      .max(4, "Code cannot be more than 4 characters"),
    isActive: Yup.boolean().required("Active is required"),
    revenueGoal: Yup.number()
      .required("Revenue Goal is required")
      .test(
        "is-decimal",
        "Revenue Goal must be a decimal with no more than two decimal places",
        (value) => {
          if (value !== undefined && value !== null) {
            return /^\d+(\.\d{1,2})?$/.test(value.toString());
          }
          return false;
        },
      ),
    deadheadGoal: Yup.number()
      .required("Deadhead Goal is required")
      .test(
        "is-decimal",
        "Deadhead Goal must be a decimal with no more than two decimal places",
        (value) => {
          if (value !== undefined && value !== null) {
            return /^\d+(\.\d{1,2})?$/.test(value.toString());
          }
          return false;
        },
      ),
    mileageGoal: Yup.number()
      .required("Mileage Goal is required")
      .test(
        "is-decimal",
        "Mileage Goal must be a decimal with no more than two decimal places",
        (value) => {
          if (value !== undefined && value !== null) {
            return /^\d+(\.\d{1,2})?$/.test(value.toString());
          }
          return false;
        },
      ),
    description: Yup.string()
      .required("Description is required")
      .max(100, "Description cannot be more than 100 characters"),
    manager: Yup.string().nullable().notRequired(),
  });
