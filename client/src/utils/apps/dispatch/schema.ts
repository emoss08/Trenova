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
import { DispatchControlFormValues } from "@/types/apps/dispatch";

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
