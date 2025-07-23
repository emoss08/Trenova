/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import * as z from "zod/v4";
import { assignmentSchema } from "./assignment-schema";
import {
  nullableIntegerSchema,
  optionalStringSchema,
  timestampSchema,
  versionSchema,
} from "./helpers";
import { stopSchema } from "./stop-schema";

export const MoveStatus = z.enum([
  "New",
  "Assigned",
  "InTransit",
  "Completed",
  "Canceled",
]);

export const moveSchema = z.object({
  id: optionalStringSchema,
  version: versionSchema,
  createdAt: timestampSchema,
  updatedAt: timestampSchema,
  organizationId: optionalStringSchema,
  businessUnitId: optionalStringSchema,

  // * Core Fields
  shipmentId: optionalStringSchema,
  status: MoveStatus,
  loaded: z.boolean(),
  sequence: z.number().min(0, { error: "Sequence cannot be negative" }),
  distance: nullableIntegerSchema,
  stops: z.array(stopSchema),
  assignment: assignmentSchema.nullish(),
  formId: optionalStringSchema, // * Simply becuase react-hook-form will override the id if there is nothing for it to append to.
});

export type MoveSchema = z.infer<typeof moveSchema>;
