import { StopStatus, StopType } from "@/types/stop";
import { type InferType, mixed, number, object, ref, string } from "yup";

export const stopSchema = object({
  id: string().optional(),
  organizationId: string().nullable().optional(),
  businessUnitId: string().nullable().optional(),
  status: mixed<StopStatus>()
    .required("Status is required")
    .oneOf(Object.values(StopStatus)),
  type: mixed<StopType>()
    .required("Type is required")
    .oneOf(Object.values(StopType)),
  sequence: number().required("Sequence is required"),
  pieces: number()
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
    .integer("Pieces must be a whole number")
    .min(0, "Pieces cannot be negative")
    .optional(),
  weight: number()
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
    .integer("Weight must be a whole number")
    .min(0, "Weight cannot be negative")
    .optional(),
  plannedArrival: number()
    .min(0, "Planned arrival cannot be negative")
    .max(ref("plannedDeparture"), "Planned arrival must be before departure")
    .required("Planned arrival is required"),
  plannedDeparture: number()
    .min(ref("plannedArrival"), "Planned departure must be after arrival")
    .required("Planned departure is required"),
  actualArrival: number()
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
    .integer("Actual arrival must be a whole number")
    .min(0, "Actual arrival cannot be negative")
    .max(ref("actualDeparture"), "Actual arrival must be before departure")
    .optional(),
  actualDeparture: number()
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
    .integer("Actual departure must be a whole number")
    .min(ref("actualArrival"), "Actual departure cannot be before arrival")
    .optional(),
  addressLine: string().required("Address line is required"),
});

export type StopSchema = InferType<typeof stopSchema>;
