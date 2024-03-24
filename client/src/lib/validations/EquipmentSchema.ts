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

import { EquipmentClassChoiceProps } from "@/lib/choices";
import { StatusChoiceProps } from "@/types";
import {
  EquipmentManufacturerFormValues,
  EquipmentStatus,
  EquipmentTypeFormValues,
  TractorFormValues,
  TrailerFormValues,
} from "@/types/equipment";
import * as Yup from "yup";
import { ObjectSchema, StringSchema } from "yup";

Yup.addMethod<StringSchema>(
  Yup.string,
  "decimal",
  function (message = "Must be a decimal") {
    return this.test("is-decimal", message, (value) => {
      if (value) {
        return /^\d+(\.\d{1,4})?$/.test(value);
      }
      return true;
    });
  },
);

export const equipmentTypeSchema: ObjectSchema<EquipmentTypeFormValues> =
  Yup.object().shape({
    status: Yup.string<StatusChoiceProps>().required("Status is required"),
    name: Yup.string()
      .required("Name is required")
      .max(50, "Name cannot be more than 50 characters"),
    description: Yup.string(),
    costPerMile: Yup.number().transform((value) =>
      Number.isNaN(value) ? undefined : value,
    ),
    equipmentClass: Yup.string<EquipmentClassChoiceProps>().required(
      "Equipment class is required",
    ),
    exemptFromTolls: Yup.boolean().required("Exempt from tolls is required"),
    fixedCost: Yup.number().transform((value) =>
      Number.isNaN(value) ? undefined : value,
    ),
    height: Yup.number().transform((value) =>
      Number.isNaN(value) ? undefined : value,
    ),
    length: Yup.number().transform((value) =>
      Number.isNaN(value) ? undefined : value,
    ),
    idlingFuelUsage: Yup.number().transform((value) =>
      Number.isNaN(value) ? undefined : value,
    ),
    weight: Yup.number().transform((value) =>
      Number.isNaN(value) ? undefined : value,
    ),
    variableCost: Yup.number().transform((value) =>
      Number.isNaN(value) ? undefined : value,
    ),
    width: Yup.number().transform((value) =>
      Number.isNaN(value) ? undefined : value,
    ),
  });

export const equipManufacturerSchema: ObjectSchema<EquipmentManufacturerFormValues> =
  Yup.object().shape({
    status: Yup.string<StatusChoiceProps>().required("Status is required"),
    name: Yup.string()
      .required("Name is required")
      .max(50, "Name cannot be more than 50 characters"),
    description: Yup.string().notRequired(),
  });

export const trailerSchema: ObjectSchema<TrailerFormValues> =
  Yup.object().shape({
    status: Yup.string<EquipmentStatus>().required("Status is required"),
    code: Yup.string().required("Code is required"),
    equipmentType: Yup.string().required("Equipment type is required"),
    manufacturer: Yup.string().notRequired(),
    make: Yup.string().notRequired(),
    model: Yup.string().notRequired(),
    year: Yup.number().notRequired(),
    vinNumber: Yup.string().notRequired(),
    fleetCode: Yup.string().notRequired(),
    state: Yup.string().notRequired(),
    licensePlateNumber: Yup.string().notRequired(),
    licensePlateState: Yup.string().notRequired(),
    lastInspection: Yup.string().notRequired(),
    registrationNumber: Yup.string().notRequired(),
    registrationState: Yup.string().notRequired(),
    registrationExpiration: Yup.string().notRequired(),
    isLeased: Yup.boolean().required("Is leased is required"),
  });

export const tractorSchema: Yup.ObjectSchema<TractorFormValues> =
  Yup.object().shape({
    status: Yup.string<EquipmentStatus>().required("Status is required"),
    code: Yup.string().required("Code is required"),
    equipmentType: Yup.string().required("Equipment type is required"),
    licensePlateNumber: Yup.string().notRequired(),
    vinNumber: Yup.string().notRequired(),
    manufacturer: Yup.string().notRequired(),
    model: Yup.string().notRequired(),
    year: Yup.number().notRequired(),
    state: Yup.string().notRequired(),
    leased: Yup.boolean().required("Leased is required"),
    leasedDate: Yup.string().notRequired(),
    primaryWorker: Yup.string().notRequired(),
    secondaryWorker: Yup.string().notRequired(),
    hosExempt: Yup.boolean().required("HOS exempt is required"),
    ownerOperated: Yup.boolean().required("Owner operated is required"),
    fleetCode: Yup.string().notRequired(),
  });
