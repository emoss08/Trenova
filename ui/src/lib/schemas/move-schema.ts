import { MoveStatus } from "@/types/move";
import * as z from "zod/v4";
import { assignmentSchema } from "./assignment-schema";
import {
  optionalStringSchema,
  timestampSchema,
  versionSchema,
} from "./helpers";
import { stopSchema } from "./stop-schema";

export const moveSchema = z.object({
  id: optionalStringSchema,
  version: versionSchema,
  createdAt: timestampSchema,
  updatedAt: timestampSchema,
  organizationId: optionalStringSchema,
  businessUnitId: optionalStringSchema,

  // * Core Fields
  // * The shipment ID will be associated on the backend
  shipmentID: optionalStringSchema,
  status: z.enum(MoveStatus),
  trailerId: optionalStringSchema,
  tractorId: optionalStringSchema,
  loaded: z.boolean(),
  sequence: z.number().min(0, { error: "Sequence cannot be negative" }),
  distance: z.preprocess(
    (value) => {
      if (value === null || value === undefined) {
        return undefined;
      }
      const parsed = parseInt(value.toString(), 10);
      return isNaN(parsed) ? undefined : parsed;
    },
    z.number().min(0, { error: "Distance cannot be negative" }),
  ),
  stops: z.array(stopSchema),
  assignment: assignmentSchema.nullable().optional(),
  formId: optionalStringSchema, // * Simply becuase react-hook-form will override the id if there is nothing for it to append to.
});

export type MoveSchema = z.infer<typeof moveSchema>;
