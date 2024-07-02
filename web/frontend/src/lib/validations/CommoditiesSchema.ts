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
