import { MoveStatus } from "@/types/move";
import { boolean, type InferType, mixed, number, object, string } from "yup";

export const moveSchema = object({
  organizationId: string().nullable().optional(),
  businessUnitId: string().nullable().optional(),
  // * The shipment ID will be associated on the backend
  shipmentID: string().optional(),
  status: mixed<MoveStatus>()
    .required("Status is required")
    .oneOf(Object.values(MoveStatus)),
  primaryWorkerId: string().required("Primary Worker is required"),
  secondaryWorkerId: string().optional(),
  trailerId: string().optional(),
  tractorId: string().optional(),
  loaded: boolean().required("Loaded is required"),
  sequence: number().required("Sequence is required"),
  distance: number()
    .transform((_, originalValue) => {
      if (
        originalValue === "" ||
        originalValue === null ||
        originalValue === undefined
      ) {
        return undefined;
      }
      const parsed = parseInt(originalValue, 10);
      return isNaN(parsed) ? undefined : parsed;
    })
    .integer("Distance must be a whole number")
    .min(0, "Distance cannot be negative"),
});

export type MoveSchema = InferType<typeof moveSchema>;
