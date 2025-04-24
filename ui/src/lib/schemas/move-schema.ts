import { MoveStatus } from "@/types/move";
import { z } from "zod";
import { stopSchema } from "./stop-schema";

export const moveSchema = z.object({
  id: z.string().optional(),
  version: z.number().optional(),
  createdAt: z.number().optional(),
  updatedAt: z.number().optional(),

  // * Core Fields
  // * The shipment ID will be associated on the backend
  shipmentID: z.string().optional(),
  status: z.nativeEnum(MoveStatus),
  trailerId: z.string().optional(),
  tractorId: z.string().optional(),
  loaded: z.boolean(),
  sequence: z.number().min(0, "Sequence cannot be negative"),
  distance: z.preprocess(
    (value) => {
      if (value === null || value === undefined) {
        return undefined;
      }
      const parsed = parseInt(value.toString(), 10);
      return isNaN(parsed) ? undefined : parsed;
    },
    z.number().min(0, "Distance cannot be negative"),
  ),
  stops: z.array(stopSchema),
  formId: z.string().optional(), // * Simply becuase react-hook-form will override the id if there is nothing for it to append to.
});

export type MoveSchema = z.infer<typeof moveSchema>;
