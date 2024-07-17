/**
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



import { type EquipmentClassChoiceProps } from "@/lib/choices";
import { type StatusChoiceProps } from "@/types";
import type {
  EquipmentManufacturerFormValues,
  EquipmentStatus,
  EquipmentTypeFormValues,
  TractorFormValues,
  TrailerFormValues,
} from "@/types/equipment";
import {
  ObjectSchema,
  StringSchema,
  addMethod,
  boolean,
  number,
  object,
  string,
} from "yup";

addMethod<StringSchema>(
  string,
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
  object().shape({
    status: string<StatusChoiceProps>().required("Status is required"),
    code: string()
      .required("Code is required")
      .max(10, "Code cannot be more than 50 characters"),
    description: string(),
    costPerMile: string().decimal().optional().nullable(),
    equipmentClass: string<EquipmentClassChoiceProps>().required(
      "Equipment class is required",
    ),
    exemptFromTolls: boolean().required("Exempt from tolls is required"),
    fixedCost: string().decimal().optional().nullable(),
    height: string().decimal().optional().nullable(),
    length: string().decimal().optional().nullable(),
    idlingFuelUsage: string().decimal().optional().nullable(),
    weight: string().decimal().optional().nullable(),
    variableCost: string().decimal().optional().nullable(),
    width: string().decimal().optional().nullable(),
    color: string().max(100, "Color cannot be more than 100 characters"),
  });

export const equipManufacturerSchema: ObjectSchema<EquipmentManufacturerFormValues> =
  object().shape({
    status: string<StatusChoiceProps>().required("Status is required"),
    name: string()
      .required("Name is required")
      .max(50, "Name cannot be more than 50 characters"),
    description: string(),
  });

export const trailerSchema: ObjectSchema<TrailerFormValues> = object().shape({
  status: string<EquipmentStatus>().required("Status is required"),
  code: string().required("Code is required"),
  equipmentTypeId: string().required("Equipment type is required"),
  equipmentManufacturerId: string().nullable(),
  make: string(),
  model: string(),
  year: number().nullable(),
  vin: string(),
  fleetCodeId: string().optional().nullable(),
  stateId: string().nullable(),
  licensePlateNumber: string(),
  lastInspectionDate: string().nullable(),
  registrationNumber: string(),
  registrationStateId: string().nullable(),
  registrationExpirationDate: string().nullable(),
});

export const tractorSchema: ObjectSchema<TractorFormValues> = object().shape({
  status: string<EquipmentStatus>().required("Status is required"),
  code: string().required("Code is required"),
  equipmentTypeId: string().required("Equipment type is required"),
  licensePlateNumber: string(),
  vin: string(),
  equipmentManufacturerId: string().required(
    "Equipment manufacturer is required",
  ),
  model: string(),
  year: number().nullable(),
  stateId: string().optional().nullable(),
  isLeased: boolean().required("Leased is required"),
  leasedDate: string().nullable(),
  primaryWorkerId: string().required("Primary worker is required"),
  secondaryWorkerId: string().nullable(),
  fleetCodeId: string().optional().nullable(),
});
