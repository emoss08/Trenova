import { z } from "zod";
import { userSchema } from "./user";

export const shipmentEventTypeSchema = z.enum([
  "ShipmentCreated",
  "ShipmentUpdated",
  "StatusChanged",
  "ShipmentCanceled",
  "ShipmentUncanceled",
  "OwnershipTransferred",
  "MoveStatusChanged",
  "MoveDeparted",
  "MoveArrived",
  "StopCompleted",
  "DriverAssigned",
  "DriverReassigned",
  "DriverUnassigned",
  "HoldPlaced",
  "HoldUpdated",
  "HoldReleased",
  "CommentPosted",
]);
export type ShipmentEventType = z.infer<typeof shipmentEventTypeSchema>;

export const shipmentEventSeveritySchema = z.enum([
  "danger",
  "success",
  "brand",
  "info",
  "muted",
]);
export type ShipmentEventSeverity = z.infer<typeof shipmentEventSeveritySchema>;

export const shipmentEventActorTypeSchema = z.enum(["user", "apikey", "system", "edi"]);
export type ShipmentEventActorType = z.infer<typeof shipmentEventActorTypeSchema>;

export const shipmentEventSchema = z.object({
  id: z.string(),
  organizationId: z.string(),
  businessUnitId: z.string(),
  shipmentId: z.string(),
  moveId: z.string().optional(),
  stopId: z.string().optional(),
  assignmentId: z.string().optional(),
  commentId: z.string().optional(),
  holdId: z.string().optional(),
  type: shipmentEventTypeSchema,
  severity: shipmentEventSeveritySchema,
  actorType: shipmentEventActorTypeSchema,
  actorId: z.string().optional(),
  actorLabel: z.string().default(""),
  summary: z.string(),
  metadata: z.record(z.string(), z.unknown()).default({}),
  occurredAt: z.number(),
  correlationId: z.string().optional(),
  actor: userSchema.partial().optional(),
  shipment: z
    .object({
      id: z.string(),
      proNumber: z.string(),
    })
    .partial()
    .optional(),
});

export type ShipmentEvent = z.infer<typeof shipmentEventSchema>;

export const shipmentEventListSchema = z.array(shipmentEventSchema);
export type ShipmentEventList = z.infer<typeof shipmentEventListSchema>;
