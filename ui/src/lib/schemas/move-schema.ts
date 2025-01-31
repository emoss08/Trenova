import { StopStatus } from "@/types/stop";
import { boolean, type InferType, mixed, number, object, string } from "yup";

export const moveSchema = object({
  id: string().optional(),
  organizationId: string().nullable().optional(),
  businessUnitId: string().nullable().optional(),
  // * The shipment ID will be associated on the backend
  shipmentID: string().optional(),
  status: mixed<StopStatus>()
    .required("Status is required")
    .oneOf(Object.values(StopStatus)),
  primaryWorkerId: string().required("Primary Worker is required"),
  secondaryWorkerId: string().optional(),
  trailerId: string().optional(),
  tractorId: string().optional(),
  loaded: boolean().required("Loaded is required"),
  sequence: number().required("Sequence is required"),
  distance: string()
    .optional()
    .transform((_, originalValue) => {
      if (
        originalValue === "" ||
        originalValue === null ||
        originalValue === undefined
      ) {
        return undefined;
      }
      const parsed = parseFloat(originalValue);
      return isNaN(parsed) ? undefined : parsed;
    }),
});

export type MoveSchema = InferType<typeof moveSchema>;
