import { Status } from "@/types/common";
import { z } from "zod";
import { customerSchema } from "./customer-schema";
import { equipmentTypeSchema } from "./equipment-type-schema";
import { locationSchema } from "./location-schema";
import { serviceTypeSchema } from "./service-type-schema";
import { shipmentTypeSchema } from "./shipment-type-schema";
import { workerSchema } from "./worker-schema";

export const dedicatedLaneSchema = z.object({
  id: z.string().optional(),
  version: z.number().optional(),
  createdAt: z.number().optional(),
  updatedAt: z.number().optional(),

  status: z.nativeEnum(Status),
  name: z
    .string()
    .min(1, "Name is required")
    .max(100, "Name must be less than 100 characters"),
  customerId: z.string().min(1, "Customer is required"),
  originLocationId: z.string().min(1, "Origin Location is required"),
  destinationLocationId: z.string().min(1, "Destination Location is required"),
  primaryWorkerId: z.string().min(1, "Primary Worker is required"),
  secondaryWorkerId: z.string().nullable().optional(),
  serviceTypeId: z.string().min(1, "Service Type is required"),
  shipmentTypeId: z.string().min(1, "Shipment Type is required"),
  tractorTypeId: z.string().nullable().optional(),
  trailerTypeId: z.string().nullable().optional(),
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
