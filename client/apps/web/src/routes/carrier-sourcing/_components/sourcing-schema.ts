import { z } from "zod";

export const CARRIER_CODE_MAX_LENGTH = 10;

export const importSourcedCarrierSchema = z.object({
  code: z.string().trim().max(CARRIER_CODE_MAX_LENGTH, "Code cannot exceed 10 characters"),
  enrollMonitoring: z.boolean(),
});

export type ImportSourcedCarrierFormValues = z.infer<typeof importSourcedCarrierSchema>;
