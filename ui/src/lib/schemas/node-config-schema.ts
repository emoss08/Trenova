import { z } from "zod";
import { ActionType, NodeType } from "./workflow-schema";

const variableStringSchema = z
  .string()
  .describe("Supports variable interpolation like {{trigger.shipmentId}}");

export const shipmentUpdateStatusConfigSchema = z.object({
  shipmentId: variableStringSchema,
  status: z.enum(["new", "in_transit", "delivered", "cancelled", "on_hold"]),
});

export const notificationSendEmailConfigSchema = z.object({
  to: variableStringSchema
    .email("Invalid email format")
    .or(variableStringSchema),
  subject: z.string().min(1, "Subject is required"),
  body: z.string().min(1, "Body is required"),
  template: z.string().optional(),
});

export const billingValidateRequirementsConfigSchema = z.object({
  shipmentId: variableStringSchema,
});

export const dataAPICallConfigSchema = z.object({
  url: z.string().url("Invalid URL").or(variableStringSchema),
  method: z.enum(["GET", "POST", "PUT", "DELETE", "PATCH"]),
  headers: z.record(z.string(), z.string()).optional(),
  body: z.string().optional(),
});

export const documentValidateCompletenessConfigSchema = z.object({
  shipmentId: variableStringSchema,
  requiredDocuments: z
    .array(z.string())
    .min(1, "At least one required document must be specified"),
});

export const actionConfigSchemas = {
  shipment_update_status: shipmentUpdateStatusConfigSchema,
  notification_send_email: notificationSendEmailConfigSchema,
  billing_validate_requirements: billingValidateRequirementsConfigSchema,
  data_api_call: dataAPICallConfigSchema,
  document_validate_completeness: documentValidateCompletenessConfigSchema,
};

export type ShipmentUpdateStatusConfig = z.infer<
  typeof shipmentUpdateStatusConfigSchema
>;
export type NotificationSendEmailConfig = z.infer<
  typeof notificationSendEmailConfigSchema
>;
export type BillingValidateRequirementsConfig = z.infer<
  typeof billingValidateRequirementsConfigSchema
>;
export type DataAPICallConfig = z.infer<typeof dataAPICallConfigSchema>;
export type DocumentValidateCompletenessConfig = z.infer<
  typeof documentValidateCompletenessConfigSchema
>;

export const conditionConfigSchema = z.object({
  field: z.string().min(1, "Field is required"),
  operator: z.enum([
    "equals",
    "notEquals",
    "contains",
    "greaterThan",
    "lessThan",
  ]),
  value: z.union([z.string(), z.number(), z.boolean()]),
});

export type ConditionConfig = z.infer<typeof conditionConfigSchema>;

export const delayConfigSchema = z.object({
  delaySeconds: z.number().min(1, "Delay must be at least 1 second"),
});

export type DelayConfig = z.infer<typeof delayConfigSchema>;

export const nodeConfigSchema = z.object({
  nodeType: NodeType,
  actionType: ActionType.optional(),
  config: z.record(z.string(), z.any()),
});

export type NodeConfigSchema = z.infer<typeof nodeConfigSchema>;
