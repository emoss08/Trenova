import {
  DistanceMethodChoiceProps,
  RouteDistanceUnitProps,
} from "@/lib/choices";
import { RouteControlFormValues } from "@/types/route";
import * as Yup from "yup";
import { ObjectSchema } from "yup";

export const routeControlSchema: ObjectSchema<RouteControlFormValues> =
  Yup.object().shape({
    distanceMethod: Yup.string<DistanceMethodChoiceProps>().required(
      "Distance method is required",
    ),
    mileageUnit: Yup.string<RouteDistanceUnitProps>().required(
      "Mileage unit is required",
    ),
    generateRoutes: Yup.boolean().required("Generate routes is required"),
  });
