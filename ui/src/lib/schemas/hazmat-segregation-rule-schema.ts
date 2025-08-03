/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { Status } from "@/types/common";
import { HazardousClassChoiceProps } from "@/types/hazardous-material";
import { SegregationType } from "@/types/hazmat-segregation-rule";
import * as z from "zod/v4";
import {
  decimalStringSchema,
  nullableStringSchema,
  optionalStringSchema,
  timestampSchema,
  versionSchema,
} from "./helpers";

export const hazmatSegregationRuleSchema = z
  .object({
    id: optionalStringSchema,
    version: versionSchema,
    createdAt: timestampSchema,
    updatedAt: timestampSchema,
    organizationId: optionalStringSchema,
    businessUnitId: optionalStringSchema,

    // * Core Fields
    status: z.enum(Status),
    name: z.string().min(1, { error: "Name is required" }),
    description: optionalStringSchema,
    classA: z.enum(HazardousClassChoiceProps),
    classB: z.enum(HazardousClassChoiceProps),
    hazmatAId: nullableStringSchema,
    hazmatBId: nullableStringSchema,
    segregationType: z.enum(SegregationType),
    minimumDistance: decimalStringSchema,
    distanceUnit: optionalStringSchema,
    hasExceptions: z.boolean(),
    exceptionNotes: optionalStringSchema,
    referenceCode: optionalStringSchema,
    regulationSource: optionalStringSchema,
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
