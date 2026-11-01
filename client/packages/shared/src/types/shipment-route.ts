import { z } from "zod";
import { stopTypeSchema } from "./shipment";

const routeLocationSchema = z.object({
  id: z.string(),
  name: z.string().nullish(),
  addressLine1: z.string().default(""),
  addressLine2: z.string().nullish(),
  city: z.string().default(""),
  postalCode: z.string().default(""),
  latitude: z.number().nullish(),
  longitude: z.number().nullish(),
  state: z.object({ id: z.string().nullish(), abbreviation: z.string().default("") }).nullish(),
});

/**
 * A shipment's moves as a record that bills it draws the route: each stop's kind and where
 * it is. Null when the reader may not see the shipment's route.
 */
export const shipmentRouteMovesSchema = z
  .array(
    z.object({
      id: z.string(),
      sequence: z.number().nullish(),
      loaded: z.boolean().nullish(),
      distance: z.number().nullish(),
      assignment: z
        .object({
          id: z.string(),
          primaryWorker: z
            .object({
              id: z.string(),
              wholeName: z.string().nullish(),
              firstName: z.string().nullish(),
              lastName: z.string().nullish(),
            })
            .nullish(),
        })
        .nullish(),
      stops: z
        .array(
          z.object({
            id: z.string(),
            type: stopTypeSchema,
            sequence: z.number().nullish(),
            actualArrival: z.number().nullish(),
            location: routeLocationSchema.nullish(),
          }),
        )
        .default([]),
    }),
  )
  .nullish();
export type ShipmentRouteMoves = z.infer<typeof shipmentRouteMovesSchema>;
