import { Status } from "@/types/common";
import { HazardousClassChoiceProps } from "@/types/hazardous-material";
import { SegregationType } from "@/types/hazmat-segregation-rule";
import { z } from "zod";

export const hazmatSegregationRuleSchema = z
  .object({
    id: z.string().optional(),
    version: z.number().optional(),
    createdAt: z.number().optional(),
    updatedAt: z.number().optional(),

    // * Core Fields
    status: z.nativeEnum(Status),
    name: z.string().min(1, "Name is required"),
    description: z.string().optional(),
    classA: z.nativeEnum(HazardousClassChoiceProps),
    classB: z.nativeEnum(HazardousClassChoiceProps),
    hazmatAId: z.string().nullable().optional(),
    hazmatBId: z.string().nullable().optional(),
    segregationType: z.nativeEnum(SegregationType),
    minimumDistance: z.preprocess((val) => {
      if (val === "" || val === null || val === undefined) {
        return undefined;
      }
      const parsed = parseFloat(String(val));
      return isNaN(parsed) ? undefined : parsed;
    }, z.number().optional()),
    distanceUnit: z.string().optional(),
    hasExceptions: z.boolean().optional(),
    exceptionNotes: z.string().optional(),
    referenceCode: z.string().optional(),
    regulationSource: z.string().optional(),
  })
  .refine(
    (data) => {
      if (data.segregationType === SegregationType.Distance) {
        return data.minimumDistance !== undefined;
      }
      return true;
    },
    {
      message: "Minimum Distance is required when segregation type is distance",
      path: ["minimumDistance"],
    },
  )
  .refine(
    (data) => {
      if (data.hasExceptions) {
        return data.exceptionNotes !== undefined;
      }
      return true;
    },
    {
      message: "Exception Notes are required when has exceptions is true",
      path: ["exceptionNotes"],
    },
  )
  .refine(
    (data) => {
      if (data.segregationType === SegregationType.Distance) {
        return data.distanceUnit !== undefined;
      }
      return true;
    },
    {
      message: "Distance Unit is required when segregation type is distance",
      path: ["distanceUnit"],
    },
  );

export type HazmatSegregationRuleSchema = z.infer<
  typeof hazmatSegregationRuleSchema
>;
