import { Status } from "@/types/common";
import { HazardousClassChoiceProps } from "@/types/hazardous-material";
import { SegregationType } from "@/types/hazmat-segregation-rule";
import { boolean, type InferType, mixed, number, object, string } from "yup";

export const hazmatSegregationRuleSchema = object({
  id: string().optional(),
  organizationId: string().nullable().optional(),
  businessUnitId: string().nullable().optional(),
  status: mixed<Status>()
    .required("Status is required")
    .oneOf(Object.values(Status)),
  name: string().required("Name is required"),
  description: string().notRequired(),
  classA: mixed<HazardousClassChoiceProps>()
    .required("Class A is required")
    .oneOf(Object.values(HazardousClassChoiceProps)),
  classB: mixed<HazardousClassChoiceProps>()
    .required("Class B is required")
    .oneOf(Object.values(HazardousClassChoiceProps)),
  hazmatAId: string().nullable().optional(),
  hazmatBId: string().nullable().optional(),
  segregationType: mixed<SegregationType>()
    .required("Segregation Type is required")
    .oneOf(Object.values(SegregationType)),
  minimumDistance: number()
    .when("segregationType", {
      is: SegregationType.Distance,
      then: (schema) => schema.required("Minimum Distance is required"),
      otherwise: (schema) => schema.nullable().notRequired(),
    })
    .transform((value) => (Number.isNaN(value) ? undefined : value)),
  distanceUnit: string().when("segregationType", {
    is: SegregationType.Distance,
    then: (schema) => schema.required("Distance Unit is required"),
    otherwise: (schema) => schema.nullable().notRequired(),
  }),
  hasExceptions: boolean().optional(),
  exceptionNotes: string().when("hasExceptions", {
    is: true,
    then: (schema) => schema.required("Exception Notes are required"),
    otherwise: (schema) => schema.notRequired(),
  }),
  referenceCode: string().optional(),
  regulationSource: string().optional(),
});

export type HazmatSegregationRuleSchema = InferType<
  typeof hazmatSegregationRuleSchema
>;
