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
  QualifierCodeFormValues,
  StopCommentFormValues,
  StopFormValues,
} from "@/types/stop";
import * as yup from "yup";

export const qualifierCodeSchema: yup.ObjectSchema<QualifierCodeFormValues> =
  yup.object().shape({
    status: yup.string<StatusChoiceProps>().required("Status is required"),
    code: yup.string().required("Name is required"),
    description: yup.string().required("Description is required"),
  });

const stopCommentSchema: yup.ObjectSchema<StopCommentFormValues> = yup
  .object()
  .shape({
    commentType: yup.string().required("Comment type is required"),
    qualifierCode: yup.string().required("Qualifier code is required"),
    comment: yup.string().required("Comment is required"),
  });

/** Stop validation schema */
export const stopSchema: yup.ObjectSchema<StopFormValues> = yup.object().shape({
  status: yup.string().required("Status is required"),
  sequence: yup.number().notRequired(),
  movement: yup.string().required("Movement is required"),
  location: yup.string().test({
    name: "location",
    test: function (value) {
      if (!value) {
        return this.parent.addressLine !== "";
      }
      return true;
    },
    message: "Stop Location is required.",
  }),
  pieces: yup.number().required("Pieces is required"),
  weight: yup.string().required("Weight is required"),
  addressLine: yup.string().test({
    name: "addressLine",
    test: function (value) {
      if (!value) {
        return this.parent.location !== "";
      }
      return true;
    },
    message: "Stop Address is required.",
  }),
  appointmentTimeWindowStart: yup
    .string()
    .required("Appointment time window start is required"),
  appointmentTimeWindowEnd: yup
    .string()
    .required("Appointment time window end is required"),
  arrivalTime: yup.string().notRequired(),
  departureTime: yup.string().notRequired(),
  stopType: yup.string().required("Stop type is required"),
  stopComments: yup.array().of(stopCommentSchema).notRequired(),
});
