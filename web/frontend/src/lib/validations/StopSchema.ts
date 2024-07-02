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
  StopTypeProps,
} from "@/types/stop";
import * as yup from "yup";
import { ShipmentStatusChoiceProps } from "../choices";

export const qualifierCodeSchema: yup.ObjectSchema<QualifierCodeFormValues> =
  yup.object().shape({
    status: yup.string<StatusChoiceProps>().required("Status is required"),
    code: yup.string().required("Name is required"),
    description: yup.string().required("Description is required"),
  });

const stopCommentSchema: yup.ObjectSchema<StopCommentFormValues> = yup
  .object()
  .shape({
    qualifierCode: yup.string().required("Qualifier code is required"),
    value: yup
      .string()
      .max(100, "Value must be less than 100 characters")
      .required("Value is required"),
  });

export const stopSchema: yup.ObjectSchema<StopFormValues> = yup.object().shape({
  status: yup
    .string<ShipmentStatusChoiceProps>()
    .required("Status is required"),
  sequence: yup.number(),
  movement: yup.string(),
  location: yup.string().test({
    name: "location-or-address",
    exclusive: false,
    message: "Either Stop Location or Stop Address is required.",
    test: function (value) {
      return (
        (value !== null && value !== "") ||
        (this.parent.addressLine !== null && this.parent.addressLine !== "")
      );
    },
  }),
  pieces: yup.number(),
  weight: yup.string().required("Weight is required"),
  addressLine: yup.string().test({
    name: "location-or-address",
    exclusive: false,
    message: "Either Stop Location or Stop Address is required.",
    test: function (value) {
      return (
        (value !== null && value !== "") ||
        (this.parent.location !== null && this.parent.location !== "")
      );
    },
  }),
  appointmentTimeWindowStart: yup
    .string()
    .required("Appointment time window start is required"),
  appointmentTimeWindowEnd: yup
    .string()
    .required("Appointment time window end is required"),
  arrivalTime: yup.string(),
  departureTime: yup.string(),
  stopType: yup.string<StopTypeProps>().required("Stop type is required"),
  stopComments: yup.array().of(stopCommentSchema),
});
