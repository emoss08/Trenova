/**
 * Copyright (c) 2024 Trenova Technologies, LLC
 *
 * Licensed under the Business Source License 1.1 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://trenova.app/pricing/
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *
 * Key Terms:
 * - Non-production use only
 * - Change Date: 2026-11-16
 * - Change License: GNU General Public License v2 or later
 *
 * For full license text, see the LICENSE file in the root directory.
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
