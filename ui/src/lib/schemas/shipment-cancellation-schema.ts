import { InferType, number, object, string } from "yup";

export const shipmentCancellationSchema = object({
  cancelReason: string().required("Cancel Reason is required"),
  shipmentId: string().required("Shipment ID is required"),
  canceledById: string().required("Canceled By is required"),
  canceledAt: number().required("Canceled At is required"),
});

export type ShipmentCancellationSchema = InferType<
  typeof shipmentCancellationSchema
>;
