import { z } from "zod";
import { nullableArraySchema } from "./helpers";

export const customerTrackingFieldSchema = z.enum([
  "Status",
  "LocationNames",
  "Windows",
  "Actuals",
  "Eta",
  "References",
  "Carrier",
  "Equipment",
  "Freight",
  "LastLocation",
  "Holds",
  "Notes",
  "ProofOfDelivery",
]);
export type CustomerTrackingFieldValue = z.infer<typeof customerTrackingFieldSchema>;

export const publicShipmentStatusSchema = z.enum([
  "Planned",
  "InTransit",
  "Delayed",
  "Delivered",
  "Canceled",
]);
export type PublicShipmentStatus = z.infer<typeof publicShipmentStatusSchema>;

/** Mirrors PublicStop in the TMS service. Each optional part is present only when shown. */
export const publicShipmentStopSchema = z.object({
  sequence: z.number().int(),
  type: z.string(),
  city: z.string(),
  stateCode: z.string(),
  locationName: z.string().optional(),
  addressLine: z.string().optional(),
  windowStart: z.number().optional(),
  windowEnd: z.number().nullable().optional(),
  arrivedAt: z.number().nullable().optional(),
  departedAt: z.number().nullable().optional(),
  estimatedArrival: z.number().nullable().optional(),
  running: z.string().optional(),
});
export type PublicShipmentStop = z.infer<typeof publicShipmentStopSchema>;

/**
 * Mirrors PublicTrackingView in the customer tracking service: what a customer
 * sees of its shipment through a link. Fields lists the parts the customer's
 * settings show; every other optional part is absent.
 */
export const publicShipmentViewSchema = z.object({
  companyName: z.string(),
  shipmentProNumber: z.string(),
  fields: nullableArraySchema(customerTrackingFieldSchema),
  expiresAt: z.number(),
  closed: z.boolean(),
  status: publicShipmentStatusSchema.optional(),
  bol: z.string().optional(),
  customerReference: z.string().optional(),
  carrierName: z.string().optional(),
  equipment: z.string().optional(),
  pieces: z.number().nullable().optional(),
  weight: z.number().nullable().optional(),
  lastLocation: z
    .object({ city: z.string(), stateCode: z.string(), reportedAt: z.number() })
    .nullable()
    .optional(),
  stops: nullableArraySchema(publicShipmentStopSchema),
  holds: z
    .array(z.object({ type: z.string(), label: z.string().optional(), startedAt: z.number() }))
    .optional(),
  notes: z.array(z.object({ text: z.string(), postedAt: z.number() })).optional(),
  proofOfDelivery: z
    .object({ documentId: z.string(), fileName: z.string(), filedAt: z.number() })
    .nullable()
    .optional(),
});
export type PublicShipmentView = z.infer<typeof publicShipmentViewSchema>;
