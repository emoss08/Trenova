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
import {
  CommodityFormValues,
  HazardousMaterialFormValues,
} from "@/types/commodities";
import { StatusChoiceProps, YesNoChoiceProps } from "@/types";
import {
  HazardousClassChoiceProps,
  PackingGroupChoiceProps,
  UnitOfMeasureChoiceProps,
} from "@/lib/choices";

export const hazardousMaterialSchema: ObjectSchema<HazardousMaterialFormValues> =
  Yup.object().shape({
    status: Yup.string<StatusChoiceProps>().required("Status is required"),
    name: Yup.string().required("Description is required"),
    description: Yup.string().notRequired(),
    hazardClass: Yup.string<HazardousClassChoiceProps>().required(
      "Hazardous Class is required",
    ),
    packingGroup: Yup.string<PackingGroupChoiceProps>().notRequired(),
    ergNumber: Yup.string<UnitOfMeasureChoiceProps>().notRequired(),
    properShippingName: Yup.string().notRequired(),
  });

export const commoditySchema: ObjectSchema<CommodityFormValues> =
  Yup.object().shape({
    status: Yup.string<StatusChoiceProps>().required("Status is required"),
    name: Yup.string()
      .max(100, "Name cannot be longer than 100 characters long.")
      .required("Name is required"),
    description: Yup.string().notRequired(),
    minTemp: Yup.string()
      .max(
        Yup.ref("maxTemp"),
        "Minimum temperature must be less than maximum temperature.",
      )
      .test(
        "is-decimal",
        "Minimum temperature cannot be more than two decimal places.",
        (value) => {
          if (value !== "" && value !== undefined) {
            return /^\d+(\.\d{1,2})?$/.test(value.toString());
          }
          return true;
        },
      )
      .notRequired(),
    maxTemp: Yup.string()
      .test(
        "is-decimal",
        "Maximum temperature cannot be more than two decimal places.",
        (value) => {
          if (value !== "" && value !== undefined) {
            return /^\d+(\.\d{1,2})?$/.test(value.toString());
          }
          return true;
        },
      )
      .notRequired(),
    setPointTemp: Yup.number().notRequired(),
    unitOfMeasure: Yup.string<UnitOfMeasureChoiceProps>().notRequired(),
    hazardousMaterial: Yup.string().notRequired(),
    isHazmat: Yup.string<YesNoChoiceProps>().required("Is Hazmat is required"),
  });
