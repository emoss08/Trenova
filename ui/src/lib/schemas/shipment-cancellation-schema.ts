import { z } from "zod";

export const shipmentCancellationSchema = z.object({
  cancelReason: z.string().min(1, "Cancel Reason is required"),
  shipmentId: z.string().min(1, "Shipment ID is required"),
  canceledById: z.string().min(1, "Canceled By is required"),
  canceledAt: z.number().min(1, "Canceled At is required"),
});

export type ShipmentCancellationSchema = z.infer<
  typeof shipmentCancellationSchema
>;

export const shipmentUncancelSchema = z.object({
  shipmentId: z.string().min(1, "Shipment ID is required"),
  updateAppointments: z.boolean().default(false),
});

export type ShipmentUncancelSchema = z.infer<typeof shipmentUncancelSchema>;
