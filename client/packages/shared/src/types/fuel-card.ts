import { z } from "zod";
import { fuelCardProviderSchema } from "./fuel-ifta-enums";

export const LAST_FOUR_PATTERN = /^\d{4}$/;

export const fuelCardEditableStatusSchema = z.enum(["Active", "Suspended"]);
export type FuelCardEditableStatus = z.infer<typeof fuelCardEditableStatusSchema>;

export const fuelCardFormSchema = z.object({
  provider: fuelCardProviderSchema,
  lastFour: z
    .string()
    .trim()
    .regex(LAST_FOUR_PATTERN, { message: "Enter the last four digits of the card" }),
  label: z
    .string()
    .trim()
    .min(1, { message: "Give the card a label" })
    .max(100, { message: "Keep the label under 100 characters" }),
  externalCardId: z
    .string()
    .max(100, { message: "Keep the token under 100 characters" })
    .nullable(),
  assignedWorkerId: z.string().nullable(),
  assignedTractorId: z.string().nullable(),
  status: fuelCardEditableStatusSchema,
  expiresAt: z.number().int().nullable(),
  notes: z.string().max(2000, { message: "Keep notes under 2000 characters" }).nullable(),
});

export type FuelCardFormValues = z.infer<typeof fuelCardFormSchema>;

export const cancelFuelCardSchema = z.object({
  reason: z
    .string()
    .trim()
    .min(10, { message: "Say why in at least ten characters; cancelling is permanent" })
    .max(500, { message: "Keep the reason under 500 characters" }),
});

export type CancelFuelCardValues = z.infer<typeof cancelFuelCardSchema>;
