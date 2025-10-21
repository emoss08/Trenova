import * as z from "zod/v4";
import { assignmentSchema } from "./assignment-schema";
import {
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
  distance: z.number().nullish(),
  stops: z.array(stopSchema),
  assignment: assignmentSchema.nullish(),
  formId: optionalStringSchema, // * Simply becuase react-hook-form will override the id if there is nothing for it to append to.
});

export type MoveSchema = z.infer<typeof moveSchema>;
