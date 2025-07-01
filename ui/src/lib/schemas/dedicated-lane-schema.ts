import { Status } from "@/types/common";
import * as z from "zod/v4";
import { customerSchema } from "./customer-schema";
import { equipmentTypeSchema } from "./equipment-type-schema";
import {
  nullableStringSchema,
  optionalStringSchema,
  timestampSchema,
  versionSchema,
} from "./helpers";
import { locationSchema } from "./location-schema";
import { serviceTypeSchema } from "./service-type-schema";
import { shipmentTypeSchema } from "./shipment-type-schema";
import { workerSchema } from "./worker-schema";

export const dedicatedLaneSchema = z.object({
  id: optionalStringSchema,
  version: versionSchema,
  createdAt: timestampSchema,
  updatedAt: timestampSchema,
  organizationId: optionalStringSchema,
  businessUnitId: optionalStringSchema,

  status: z.enum(Status),
  name: z
    .string()
    .min(1, { error: "Name is required" })
    .max(100, { error: "Name must be less than 100 characters" }),
  customerId: z.string().min(1, { error: "Customer is required" }),
  originLocationId: z.string().min(1, { error: "Origin Location is required" }),
  destinationLocationId: z
    .string()
    .min(1, { error: "Destination Location is required" }),
  primaryWorkerId: z.string().min(1, { error: "Primary Worker is required" }),
  serviceTypeId: z.string().min(1, { error: "Service Type is required" }),
  shipmentTypeId: z.string().min(1, { error: "Shipment Type is required" }),
  tractorTypeId: nullableStringSchema,
  trailerTypeId: nullableStringSchema,
  secondaryWorkerId: nullableStringSchema,
  autoAssign: z.boolean(),

  shipmentType: shipmentTypeSchema.nullable().optional(),
  serviceType: serviceTypeSchema.nullable().optional(),
  tractorType: equipmentTypeSchema.nullable().optional(),
  trailerType: equipmentTypeSchema.nullable().optional(),
  customer: customerSchema.nullable().optional(),
  originLocation: locationSchema.nullable().optional(),
  destinationLocation: locationSchema.nullable().optional(),
  primaryWorker: workerSchema.nullable().optional(),
  secondaryWorker: workerSchema.nullable().optional(),
});

export type DedicatedLaneSchema = z.infer<typeof dedicatedLaneSchema>;

export enum SuggestionStatus {
  Pending = "Pending",
  Accepted = "Accepted",
  Rejected = "Rejected",
  Expired = "Expired",
}

export const dedicatedLaneSuggestionSchema = z.object({
  id: z.string().optional(),
  version: z.number().optional(),
  createdAt: z.number().optional(),
  updatedAt: z.number().optional(),

  status: z.enum(SuggestionStatus),
  customerId: z.string().min(1, "Customer is required"),
  originLocationId: z.string().min(1, "Origin Location is required"),
  destinationLocationId: z.string().min(1, "Destination Location is required"),
  serviceTypeId: z.string().min(1, "Service Type is required"),
  shipmentTypeId: z.string().min(1, "Shipment Type is required"),
  trailerTypeId: z.string().nullable().optional(),
  tractorTypeId: z.string().nullable().optional(),
  confidenceScore: z
    .string()
    .min(0, "Confidence Score must be greater than 0")
    .transform((val) => {
      if (val == "" || val === null || val === undefined) {
        return undefined;
      }
      return val;
    }),
  frequencyCount: z.number().min(0, "Frequency Count must be greater than 0"),
  averageFreightCharge: z
    .string()
    .min(0, "Average Freight Charge must be greater than 0")
    .transform((val) => {
      if (val == "" || val === null || val === undefined) {
        return undefined;
      }
      return val;
    }),
  totalFreightValue: z
    .string()
    .min(0, "Total Freight Value must be greater than 0")
    .transform((val) => {
      if (val == "" || val === null || val === undefined) {
        return undefined;
      }
    }),
  lastShipmentDate: z
    .number()
    .min(0, "Last Shipment Date must be greater than 0"),
  firstShipmentDate: z
    .number()
    .min(0, "First Shipment Date must be greater than 0"),
  analysisStartDate: z
    .number()
    .min(0, "Analysis Start Date must be greater than 0"),
  analysisEndDate: z
    .number()
    .min(0, "Analysis End Date must be greater than 0"),
  suggestedName: z.string().min(1, "Suggested Name is required"),
  patternDetails: z.record(z.string(), z.any()),
  createdDedicatedLaneId: z.string().nullable().optional(),
  processedById: z.string().nullable().optional(),
  processedAt: z.number().nullable().optional(),
  expiredAt: z.number().nullable().optional(),

  customer: customerSchema.nullable().optional(),
  originLocation: locationSchema.nullable().optional(),
  destinationLocation: locationSchema.nullable().optional(),
  serviceType: serviceTypeSchema.nullable().optional(),
  shipmentType: shipmentTypeSchema.nullable().optional(),
  tractorType: equipmentTypeSchema.nullable().optional(),
  trailerType: equipmentTypeSchema.nullable().optional(),
  createdDedicatedLane: dedicatedLaneSchema.nullable().optional(),
});

export type DedicatedLaneSuggestionSchema = z.infer<
  typeof dedicatedLaneSuggestionSchema
>;

export const suggestionAcceptRequestSchema = z.object({
  id: z.string(),
  dedicatedLaneName: z.string().optional(),
  primaryWorkerId: z.string().optional(),
  secondaryWorkerId: z.string().optional(),
  autoAssign: z.boolean().optional(),
});

export type SuggestionAcceptRequestSchema = z.infer<
  typeof suggestionAcceptRequestSchema
>;

export const suggestionRejectRequestSchema = z.object({
  id: z.string(),
  rejectReason: z.string().optional(),
});

export type SuggestionRejectRequestSchema = z.infer<
  typeof suggestionRejectRequestSchema
>;
