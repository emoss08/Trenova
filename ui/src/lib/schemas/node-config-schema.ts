import { z } from "zod";
import { HttpMethod } from "./common-schema";
import { optionalStringSchema } from "./helpers";
import { ShipmentStatus } from "./shipment-schema";
import { ActionType, NodeType } from "./workflow-schema";

const variableStringSchema = z
  .string()
  .min(1, { error: "Shipment ID is required" })
  .describe("Supports variable interpolation like {{trigger.shipmentId}}");

const variableEmailSchema = z
  .email()
  .describe("Supports variable interpolation like {{trigger.customer.email}}");

const Operator = z.enum([
  "equals",
  "notEquals",
  "contains",
  "greaterThan",
  "lessThan",
]);

export const shipmentUpdateStatusConfigSchema = z.object({
  shipmentId: variableStringSchema,
  status: ShipmentStatus,
});

export const notificationSendEmailConfigSchema = z.object({
  to: variableEmailSchema.or(variableStringSchema),
  subject: z.string().min(1, { error: "Subject is required" }),
  body: z.string().min(1, { error: "Body is required" }),
  template: optionalStringSchema,
});

export const billingValidateRequirementsConfigSchema = z.object({
  shipmentId: variableStringSchema,
});

export const dataAPICallConfigSchema = z.object({
  url: z.url({ error: "Invalid URL" }).or(variableStringSchema),
  method: HttpMethod,
  headers: z.array(
    z.object({
      key: z.string(),
      value: z.string(),
    }),
  ),
  body: optionalStringSchema,
});

export const documentValidateCompletenessConfigSchema = z.object({
  shipmentId: variableStringSchema,
  requiredDocuments: z
    .array(z.object({ value: z.string() }))
    .min(1, { error: "At least one required document must be specified" }),
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
  field: z.string().min(1, { error: "Field is required" }),
  operator: Operator,
  value: z.union([z.string(), z.number(), z.boolean()]),
});

export type ConditionConfig = z.infer<typeof conditionConfigSchema>;

export const delayConfigSchema = z.object({
  delaySeconds: z.number().min(1, { error: "Delay must be at least 1 second" }),
});

export type DelayConfig = z.infer<typeof delayConfigSchema>;

export const nodeConfigSchema = z.object({
  nodeType: NodeType,
  actionType: ActionType.optional(),
  config: z.record(z.string(), z.any()),
});

export type NodeConfigSchema = z.infer<typeof nodeConfigSchema>;

export type OperatorSchema = z.infer<typeof Operator>;

export type ShipmentUpdateStatusConfigSchema = z.infer<
  typeof shipmentUpdateStatusConfigSchema
>;
