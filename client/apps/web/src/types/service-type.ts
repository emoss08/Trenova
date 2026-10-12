import { z } from "zod";
import {
  optionalStringSchema,
  statusSchema,
  tenantInfoSchema,
} from "@trenova/shared/types/helpers";
import { translate } from "@trenova/shared/i18n/runtime";

export const serviceTypeSchema = z.object({
  ...tenantInfoSchema.shape,
  status: statusSchema,
  code: z.string().min(1, { error: () => translate("Code is required") }),
  description: optionalStringSchema,
  color: optionalStringSchema,
});

export type ServiceType = z.infer<typeof serviceTypeSchema>;

export const bulkUpdateServiceTypeStatusRequestSchema = z.object({
  serviceTypeIds: z.array(z.string()),
  status: statusSchema,
});

export type BulkUpdateServiceTypeStatusRequest = z.infer<
  typeof bulkUpdateServiceTypeStatusRequestSchema
>;

export const bulkUpdateServiceTypeStatusResponseSchema = z.array(serviceTypeSchema);

export type BulkUpdateServiceTypeStatusResponse = z.infer<
  typeof bulkUpdateServiceTypeStatusResponseSchema
>;
