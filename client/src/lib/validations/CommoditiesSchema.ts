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

import {
  HazardousClassChoiceProps,
  PackingGroupChoiceProps,
  UnitOfMeasureChoiceProps,
} from "@/lib/choices";
import { StatusChoiceProps } from "@/types";
import {
  CommodityFormValues,
  HazardousMaterialFormValues,
} from "@/types/commodities";
import * as Yup from "yup";
import { ObjectSchema } from "yup";

export const hazardousMaterialSchema: ObjectSchema<HazardousMaterialFormValues> =
  Yup.object().shape({
    status: Yup.string<StatusChoiceProps>().required("Status is required"),
    name: Yup.string().required("Description is required"),
    description: Yup.string(),
    hazardClass: Yup.string<HazardousClassChoiceProps>().required(
      "Hazardous Class is required",
    ),
    packingGroup: Yup.string<PackingGroupChoiceProps>(),
    ergNumber: Yup.string<UnitOfMeasureChoiceProps>(),
    properShippingName: Yup.string(),
  });

export const commoditySchema: ObjectSchema<CommodityFormValues> =
  Yup.object().shape({
    status: Yup.string<StatusChoiceProps>().required("Status is required"),
    name: Yup.string()
      .max(100, "Name cannot be longer than 100 characters long.")
      .required("Name is required"),
    description: Yup.string(),
    minTemp: Yup.number()
      .max(
        Yup.ref("maxTemp"),
        "Minimum temperature must be less than maximum temperature.",
      )
      .transform((value) => (Number.isNaN(value) ? undefined : value)),
    maxTemp: Yup.number().transform((value) =>
      Number.isNaN(value) ? undefined : value,
    ),
    unitOfMeasure: Yup.string<UnitOfMeasureChoiceProps>(),
    hazardousMaterialId: Yup.string().nullable(),
    isHazmat: Yup.boolean().required("Is Hazmat is required"),
  });
