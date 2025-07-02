import * as z from "zod/v4";
import {
  decimalStringSchema,
  optionalStringSchema,
  timestampSchema,
  versionSchema,
} from "./helpers";
import { shipmentSchema } from "./shipment-schema";
import { userSchema } from "./user-schema";

export enum ConsolidationStatus {
  New = "New",
  InProgress = "InProgress",
  Completed = "Completed",
  Canceled = "Canceled",
}

export const consolidationGroupSchema = z.object({
  id: optionalStringSchema,
  version: versionSchema,
  createdAt: timestampSchema,
  updatedAt: timestampSchema,
  organizationId: optionalStringSchema,
  businessUnitId: optionalStringSchema,

  // Basic Information
  name: z.string().min(1, "Name is required"),
  description: optionalStringSchema,
  status: z.enum(ConsolidationStatus).default(ConsolidationStatus.New),
  consolidationNumber: optionalStringSchema, // * Auto-generated consolidation code

  // Consolidation Details
  // totalShipments: nullableIntegerSchema,
  // totalWeight: decimalStringSchema,
  // totalVolume: decimalStringSchema,
  // totalPallets: nullableIntegerSchema,
  // totalPieces: nullableIntegerSchema,

  // // Route Information
  // originLocationId: optionalStringSchema,
  // destinationLocationId: optionalStringSchema,
  // plannedPickupDate: timestampSchema,
  // plannedDeliveryDate: timestampSchema,
  // actualPickupDate: timestampSchema.nullish(),
  // actualDeliveryDate: timestampSchema.nullish(),

  // // Financial Information
  // estimatedCost: decimalStringSchema,
  // actualCost: decimalStringSchema,
  // estimatedSavings: decimalStringSchema,

  // // Completion Information
  // completedById: optionalStringSchema,
  // completedBy: userSchema.optional(),
  // completedAt: timestampSchema.nullish(),

  // Cancellation Information
  canceledById: optionalStringSchema,
  canceledBy: userSchema.optional(),
  canceledAt: timestampSchema.nullable(),
  cancelReason: optionalStringSchema,

  // Related Data
  shipments: z.array(shipmentSchema).optional(),

  // Metrics
  routeEfficiencyScore: decimalStringSchema,
  consolidationScore: decimalStringSchema,

  // // Notes
  // notes: optionalStringSchema,
  // internalNotes: optionalStringSchema,
});

export type ConsolidationGroupSchema = z.infer<typeof consolidationGroupSchema>;

// Schema for creating a new consolidation
export const createConsolidationSchema = z.object({
  shipmentIds: z.array(z.string()).min(2, "At least 2 shipments are required"),
  // notes: optionalStringSchema,
});

export type CreateConsolidationSchema = z.infer<
  typeof createConsolidationSchema
>;

// Schema for updating a consolidation
export const updateConsolidationSchema = consolidationGroupSchema
  .partial()
  .omit({
    id: true,
    version: true,
    createdAt: true,
    updatedAt: true,
    organizationId: true,
    businessUnitId: true,
  });

export type UpdateConsolidationSchema = z.infer<
  typeof updateConsolidationSchema
>;
