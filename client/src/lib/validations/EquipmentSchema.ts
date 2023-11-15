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
import { ObjectSchema, StringSchema } from "yup";
import {
  EquipmentManufacturerFormValues,
  EquipmentTypeFormValues,
} from "@/types/equipment";
import { EquipmentClassChoiceProps } from "@/lib/choices";
import { StatusChoiceProps } from "@/types";

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
    description: Yup.string().notRequired(),
    costPerMile: Yup.string()
      .decimal("Cost per mile must be a decimal")
      .notRequired(),
    equipmentClass: Yup.string<EquipmentClassChoiceProps>().required(
      "Equipment class is required",
    ),
    exemptFromTolls: Yup.boolean().required("Exempt from tolls is required"),
    fixedCost: Yup.string()
      .decimal("Fixed Cost must be a decimal")
      .nullable()
      .notRequired(),
    height: Yup.string()
      .decimal("Height must be a decimal")
      .nullable()
      .notRequired(),
    length: Yup.string().decimal("Length must be a decimal").notRequired(),
    idlingFuelUsage: Yup.string()
      .decimal("Idling fuel usage must be a decimal")
      .notRequired(),
    weight: Yup.string().decimal("Weight must be a decimal").notRequired(),
    variableCost: Yup.string()
      .decimal("Variable Cost must be a decimal")
      .notRequired(),
    width: Yup.string().decimal("Width must be a decimal").notRequired(),
  });

export const equipManufacturerSchema: ObjectSchema<EquipmentManufacturerFormValues> =
  Yup.object().shape({
    status: Yup.string<StatusChoiceProps>().required("Status is required"),
    name: Yup.string()
      .required("Name is required")
      .max(50, "Name cannot be more than 50 characters"),
    description: Yup.string().notRequired(),
  });
