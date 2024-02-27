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

import * as yup from "yup";
import {
  ShipmentControlFormValues,
  ReasonCodeFormValues,
  ServiceTypeFormValues,
  ShipmentTypeFormValues,
} from "@/types/shipment";

export const shipmentControlSchema: yup.ObjectSchema<ShipmentControlFormValues> =
  yup.object().shape({
    autoRateShipment: yup.boolean().required("Auto Rate Shipments is required"),
    calculateDistance: yup.boolean().required("Calculate Distance is required"),
    enforceRevCode: yup.boolean().required("Enforce Rev Code is required"),
    enforceVoidedComm: yup
      .boolean()
      .required("Enforce Voided Comm is required"),
    generateRoutes: yup.boolean().required("Generate Routes is required"),
    enforceCommodity: yup.boolean().required("Enforce Commodity is required"),
    autoSequenceStops: yup
      .boolean()
      .required("Auto Sequence Stops is required"),
    autoShipmentTotal: yup
      .boolean()
      .required("Auto Shipment Total is required"),
    enforceOriginDestination: yup
      .boolean()
      .required("Enforce Origin Destination is required"),
    checkForDuplicateBol: yup
      .boolean()
      .required("Check for Duplicate BOL is required"),
    removeShipment: yup.boolean().required("Remove Shipment is required"),
  });

import { CodeTypeProps } from "@/lib/choices";
import { StatusChoiceProps } from "@/types";

export const serviceTypeSchema: yup.ObjectSchema<ServiceTypeFormValues> = yup
  .object()
  .shape({
    status: yup.string<StatusChoiceProps>().required("Status is required"),
    code: yup
      .string()
      .max(10, "Code must be at most 10 characters")
      .required("Code is required"),
    description: yup.string().notRequired(),
  });

export const reasonCodeSchema: yup.ObjectSchema<ReasonCodeFormValues> = yup
  .object()
  .shape({
    status: yup.string<StatusChoiceProps>().required("Status is required"),
    code: yup
      .string()
      .max(10, "Code must be at most 10 characters")
      .required("Code is required"),
    codeType: yup.string<CodeTypeProps>().required("Code type is required"),
    description: yup.string().required("Description is required"),
  });

export const shipmentTypeSchema: yup.ObjectSchema<ShipmentTypeFormValues> = yup
  .object()
  .shape({
    status: yup.string<StatusChoiceProps>().required("Status is required"),
    code: yup
      .string()
      .max(10, "Name must be at most 100 characters")
      .required("Code is required"),
    description: yup.string().notRequired(),
  });
