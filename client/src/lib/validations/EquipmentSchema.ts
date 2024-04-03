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
    code: Yup.string()
      .required("Code is required")
      .max(10, "Code cannot be more than 50 characters"),
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
    color: Yup.string().max(100, "Color cannot be more than 100 characters"),
  });

export const equipManufacturerSchema: ObjectSchema<EquipmentManufacturerFormValues> =
  Yup.object().shape({
    status: Yup.string<StatusChoiceProps>().required("Status is required"),
    name: Yup.string()
      .required("Name is required")
      .max(50, "Name cannot be more than 50 characters"),
    description: Yup.string(),
  });

export const trailerSchema: ObjectSchema<TrailerFormValues> =
  Yup.object().shape({
    status: Yup.string<EquipmentStatus>().required("Status is required"),
    code: Yup.string().required("Code is required"),
    equipmentTypeId: Yup.string().required("Equipment type is required"),
    equipmentManufacturerId: Yup.string().nullable(),
    make: Yup.string(),
    model: Yup.string(),
    year: Yup.number().nullable(),
    vin: Yup.string(),
    fleetCodeId: Yup.string().required("Fleet code is required"),
    stateId: Yup.string().nullable(),
    licensePlateNumber: Yup.string(),
    lastInspectionDate: Yup.string(),
    registrationNumber: Yup.string(),
    registrationStateId: Yup.string().nullable(),
    registrationExpirationDate: Yup.string().nullable(),
  });

export const tractorSchema: Yup.ObjectSchema<TractorFormValues> =
  Yup.object().shape({
    status: Yup.string<EquipmentStatus>().required("Status is required"),
    code: Yup.string().required("Code is required"),
    equipmentTypeId: Yup.string().required("Equipment type is required"),
    licensePlateNumber: Yup.string(),
    vin: Yup.string(),
    equipmentManufacturerId: Yup.string().nullable(),
    model: Yup.string(),
    year: Yup.number().nullable(),
    state: Yup.string(),
    leased: Yup.boolean().required("Leased is required"),
    leasedDate: Yup.string().nullable(),
    primaryWorkerId: Yup.string().required("Primary worker is required"),
    secondaryWorkerId: Yup.string().nullable(),
    fleetCodeId: Yup.string().required("Fleet code is required"),
  });
