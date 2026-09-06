import { z } from "zod";
import { userSchema } from "@trenova/shared/types/user";
import { holdSeveritySchema, holdTypeSchema } from "@/types/hold-reason";
import {
  commentPriorityEnum,
  commentTypeEnum,
  commentVisibilityEnum,
} from "@/types/shipment-comment";

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
  "CarrierAssigned",
  "CarrierUnassigned",
  "TenderOffered",
  "TenderAccepted",
  "TenderDeclined",
  "TenderExpired",
  "TenderWithdrawn",
  "TenderNeedsReview",
  "RoutingGuideExhausted",
  "TenderLateResponse",
  "TenderDeliveryFailed",
  "TenderEntrySkipped",
  "TenderEntryWarned",
  "HoldPlaced",
  "HoldUpdated",
  "HoldReleased",
  "CommentPosted",
]);
export type ShipmentEventType = z.infer<typeof shipmentEventTypeSchema>;

export const shipmentEventSeveritySchema = z.enum(["danger", "success", "brand", "info", "muted"]);
export type ShipmentEventSeverity = z.infer<typeof shipmentEventSeveritySchema>;

export const shipmentEventActorTypeSchema = z.enum(["user", "apikey", "system", "edi"]);
export type ShipmentEventActorType = z.infer<typeof shipmentEventActorTypeSchema>;

export const shipmentEventTypenameSchema = z.enum([
  "ShipmentLifecycleEvent",
  "ShipmentOwnershipEvent",
  "ShipmentMoveEvent",
  "ShipmentAssignmentEvent",
  "ShipmentCarrierEvent",
  "ShipmentTenderEvent",
  "ShipmentHoldEvent",
  "ShipmentCommentEvent",
]);
export type ShipmentEventTypename = z.infer<typeof shipmentEventTypenameSchema>;

function optionalGraphQL<T extends z.ZodType>(schema: T) {
  return schema
    .nullable()
    .transform((value) => value ?? undefined)
    .optional();
}

const optionalGraphQLStringSchema = optionalGraphQL(z.string());
const optionalGraphQLIntSchema = optionalGraphQL(z.number().int());
const graphQLStringListSchema = z.array(z.string());

const shipmentEventEnvelopeSchema = z.object({
  id: z.string(),
  organizationId: z.string(),
  businessUnitId: z.string(),
  shipmentId: z.string(),
  type: shipmentEventTypeSchema,
  severity: shipmentEventSeveritySchema,
  actorType: shipmentEventActorTypeSchema,
  actorId: optionalGraphQLStringSchema,
  actorLabel: z.string().default(""),
  summary: z.string(),
  metadata: z.record(z.string(), z.unknown()).default({}),
  occurredAt: z.number(),
  correlationId: optionalGraphQLStringSchema,
  actor: optionalGraphQL(userSchema.partial()),
  shipment: optionalGraphQL(
    z
      .object({
        id: optionalGraphQLStringSchema,
        proNumber: optionalGraphQLStringSchema,
      })
      .partial(),
  ),
});

export const shipmentLifecycleEventSchema = shipmentEventEnvelopeSchema.extend({
  __typename: z.literal("ShipmentLifecycleEvent"),
  proNumber: optionalGraphQLStringSchema,
  previousStatus: optionalGraphQLStringSchema,
  newStatus: optionalGraphQLStringSchema,
  reason: optionalGraphQLStringSchema,
});
export type ShipmentLifecycleEvent = z.infer<typeof shipmentLifecycleEventSchema>;

export const shipmentOwnershipEventSchema = shipmentEventEnvelopeSchema.extend({
  __typename: z.literal("ShipmentOwnershipEvent"),
  proNumber: optionalGraphQLStringSchema,
  previousOwnerId: optionalGraphQLStringSchema,
  newOwnerId: optionalGraphQLStringSchema,
});
export type ShipmentOwnershipEvent = z.infer<typeof shipmentOwnershipEventSchema>;

export const shipmentMoveEventSchema = shipmentEventEnvelopeSchema.extend({
  __typename: z.literal("ShipmentMoveEvent"),
  moveId: optionalGraphQLStringSchema,
  stopId: optionalGraphQLStringSchema,
  previousStatus: optionalGraphQLStringSchema,
  newStatus: optionalGraphQLStringSchema,
});
export type ShipmentMoveEvent = z.infer<typeof shipmentMoveEventSchema>;

export const shipmentAssignmentEventSchema = shipmentEventEnvelopeSchema.extend({
  __typename: z.literal("ShipmentAssignmentEvent"),
  moveId: optionalGraphQLStringSchema,
  assignmentId: optionalGraphQLStringSchema,
  primaryWorkerId: optionalGraphQLStringSchema,
  secondaryWorkerId: optionalGraphQLStringSchema,
  tractorId: optionalGraphQLStringSchema,
  trailerId: optionalGraphQLStringSchema,
  driverName: optionalGraphQLStringSchema,
});
export type ShipmentAssignmentEvent = z.infer<typeof shipmentAssignmentEventSchema>;

export const shipmentCarrierEventSchema = shipmentEventEnvelopeSchema.extend({
  __typename: z.literal("ShipmentCarrierEvent"),
  moveId: optionalGraphQLStringSchema,
  carrierId: optionalGraphQLStringSchema,
  carrierName: optionalGraphQLStringSchema,
  totalCost: optionalGraphQLStringSchema,
  reason: optionalGraphQLStringSchema,
  proNumber: optionalGraphQLStringSchema,
});
export type ShipmentCarrierEvent = z.infer<typeof shipmentCarrierEventSchema>;

export const shipmentTenderEventSchema = shipmentEventEnvelopeSchema.extend({
  __typename: z.literal("ShipmentTenderEvent"),
  tenderId: optionalGraphQLStringSchema,
  offerId: optionalGraphQLStringSchema,
  moveId: optionalGraphQLStringSchema,
  carrierName: optionalGraphQLStringSchema,
  rank: optionalGraphQLIntSchema,
  channel: optionalGraphQLStringSchema,
  source: optionalGraphQLStringSchema,
  reason: optionalGraphQLStringSchema,
  action: optionalGraphQLStringSchema,
  mode: optionalGraphQLStringSchema,
  error: optionalGraphQLStringSchema,
  reasons: graphQLStringListSchema,
  warnings: graphQLStringListSchema,
});
export type ShipmentTenderEvent = z.infer<typeof shipmentTenderEventSchema>;

export const shipmentHoldEventSchema = shipmentEventEnvelopeSchema.extend({
  __typename: z.literal("ShipmentHoldEvent"),
  holdId: optionalGraphQLStringSchema,
  holdType: optionalGraphQL(holdTypeSchema),
  holdSeverity: optionalGraphQL(holdSeveritySchema),
  holdSource: optionalGraphQLStringSchema,
});
export type ShipmentHoldEvent = z.infer<typeof shipmentHoldEventSchema>;

export const shipmentCommentEventSchema = shipmentEventEnvelopeSchema.extend({
  __typename: z.literal("ShipmentCommentEvent"),
  commentId: optionalGraphQLStringSchema,
  commentBody: optionalGraphQLStringSchema,
  commentType: optionalGraphQL(commentTypeEnum),
  commentVisibility: optionalGraphQL(commentVisibilityEnum),
  commentPriority: optionalGraphQL(commentPriorityEnum),
  mentionedUserIds: graphQLStringListSchema,
});
export type ShipmentCommentEvent = z.infer<typeof shipmentCommentEventSchema>;

export const shipmentEventSchema = z.discriminatedUnion("__typename", [
  shipmentLifecycleEventSchema,
  shipmentOwnershipEventSchema,
  shipmentMoveEventSchema,
  shipmentAssignmentEventSchema,
  shipmentCarrierEventSchema,
  shipmentTenderEventSchema,
  shipmentHoldEventSchema,
  shipmentCommentEventSchema,
]);
export type ShipmentEvent = z.infer<typeof shipmentEventSchema>;

export const shipmentEventListSchema = z.array(shipmentEventSchema);
export type ShipmentEventList = z.infer<typeof shipmentEventListSchema>;
