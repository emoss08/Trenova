import { z } from "zod";
import { translate } from "@trenova/shared/i18n/runtime";

export const CARRIER_CODE_MAX_LENGTH = 10;

export const importSourcedCarrierSchema = z.object({
  code: z
    .string()
    .trim()
    .max(CARRIER_CODE_MAX_LENGTH, { error: () => translate("Code cannot exceed 10 characters") }),
  enrollMonitoring: z.boolean(),
});

export type ImportSourcedCarrierFormValues = z.infer<typeof importSourcedCarrierSchema>;
