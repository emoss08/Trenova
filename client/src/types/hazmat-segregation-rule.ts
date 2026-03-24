import { z } from "zod";
import { hazardousClassSchema } from "./hazardous-material";
import {
  decimalStringSchema,
  nullableStringSchema,
  optionalStringSchema,
  statusSchema,
  tenantInfoSchema,
} from "./helpers";

export const segregationTypeSchema = z.enum([
  "Prohibited",
  "Separated",
  "Distance",
  "Barrier",
]);

export type SegregationType = z.infer<typeof segregationTypeSchema>;

export const segregationDistanceUnitSchema = z.enum(["FT", "M", "IN", "CM"]);

export type SegregationDistanceUnit = z.infer<
  typeof segregationDistanceUnitSchema
>;

export const hazmatSegregationRuleSchema = z
  .object({
    ...tenantInfoSchema.shape,
    status: statusSchema,
    name: z.string().min(1, { message: "Name is required" }),
    description: optionalStringSchema,
    classA: hazardousClassSchema,
    classB: hazardousClassSchema,
    hazmatAId: nullableStringSchema,
    hazmatBId: nullableStringSchema,
    segregationType: segregationTypeSchema,
    minimumDistance: decimalStringSchema,
    distanceUnit: optionalStringSchema,
    hasExceptions: z.boolean().default(false),
    exceptionNotes: optionalStringSchema,
    referenceCode: optionalStringSchema,
    regulationSource: optionalStringSchema,
  })
  .refine(
    (data) =>
      data.segregationType !== "Distance" ||
      (typeof data.minimumDistance === "number" && data.minimumDistance > 0),
    {
      message: "Minimum distance is required when segregation type is Distance",
      path: ["minimumDistance"],
    },
  )
  .refine(
    (data) =>
      data.segregationType !== "Distance" ||
      (typeof data.distanceUnit === "string" && data.distanceUnit.length > 0),
    {
      message: "Distance unit is required when segregation type is Distance",
      path: ["distanceUnit"],
    },
  )
  .refine(
    (data) =>
      !data.hasExceptions ||
      (typeof data.exceptionNotes === "string" &&
        data.exceptionNotes.length > 0),
    {
      message: "Exception notes are required when has exceptions is true",
      path: ["exceptionNotes"],
    },
  );

export type HazmatSegregationRule = z.infer<typeof hazmatSegregationRuleSchema>;
