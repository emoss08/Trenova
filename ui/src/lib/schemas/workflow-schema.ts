/**
 * Workflow Automation Schemas
 *
 * Zod schemas for workflow entities matching the backend Go domain models
 */
import * as z from "zod";
import {
  nullablePulidSchema,
  optionalStringSchema,
  pulidSchema,
  timestampSchema,
  versionSchema,
} from "./helpers";

// ==================== Enums ====================

export const WorkflowStatus = z.enum([
  "draft",
  "active",
  "inactive",
  "archived",
]);

export type WorkflowStatusType = z.infer<typeof WorkflowStatus>;

export const TriggerType = z.enum([
  "manual",
  "scheduled",
  "shipment_status",
  "document_uploaded",
  "entity_created",
  "entity_updated",
  "webhook",
]);

export type TriggerTypeType = z.infer<typeof TriggerType>;

export const NodeType = z.enum([
  "trigger",
  "action",
  "condition",
  "loop",
  "delay",
  "end",
]);

export type NodeTypeType = z.infer<typeof NodeType>;

export const ActionType = z.enum([
  // Shipment actions
  "shipment_update_status",
  "shipment_assign_carrier",
  "shipment_assign_driver",
  "shipment_update_field",
  // Billing actions
  "billing_validate_requirements",
  "billing_transfer_to_queue",
  "billing_generate_invoice",
  "billing_send_invoice",
  // Document actions
  "document_validate_completeness",
  "document_request_missing",
  "document_generate",
  // Notification actions
  "notification_send_email",
  "notification_send_sms",
  "notification_send_webhook",
  "notification_send_push",
  // Data actions
  "data_transform",
  "data_api_call",
  "data_database_query",
  // Flow control actions
  "flow_approval_request",
  "flow_wait_for_event",
  "flow_parallel_execution",
]);

export type ActionTypeType = z.infer<typeof ActionType>;

export const ExecutionStatus = z.enum([
  "pending",
  "running",
  "completed",
  "failed",
  "cancelled",
  "timeout",
]);

export type ExecutionStatusType = z.infer<typeof ExecutionStatus>;

// ==================== Trigger Configurations ====================

export const scheduledTriggerConfigSchema = z.object({
  cronExpression: z.string().min(1, "Cron expression is required"),
  timezone: optionalStringSchema,
});

export type ScheduledTriggerConfigSchema = z.infer<
  typeof scheduledTriggerConfigSchema
>;

export const shipmentStatusTriggerConfigSchema = z.object({
  statuses: z.array(z.string()).min(1, "At least one status is required"),
});

export type ShipmentStatusTriggerConfigSchema = z.infer<
  typeof shipmentStatusTriggerConfigSchema
>;

export const documentUploadTriggerConfigSchema = z.object({
  documentTypes: z
    .array(z.string())
    .min(1, "At least one document type is required"),
  entityTypes: z.array(z.string()).optional(),
});

export type DocumentUploadTriggerConfigSchema = z.infer<
  typeof documentUploadTriggerConfigSchema
>;

export const entityEventTriggerConfigSchema = z.object({
  entityType: z.string().min(1, "Entity type is required"),
});

export type EntityEventTriggerConfigSchema = z.infer<
  typeof entityEventTriggerConfigSchema
>;

export const webhookTriggerConfigSchema = z.object({
  webhookUrl: optionalStringSchema,
  requireAuth: z.boolean(),
  authToken: optionalStringSchema,
  customHeaders: z.record(z.string(), z.string()).optional(),
});

export type WebhookTriggerConfigSchema = z.infer<
  typeof webhookTriggerConfigSchema
>;

export const manualTriggerConfigSchema = z.object({
  requireConfirmation: z.boolean(),
  confirmationMessage: optionalStringSchema,
});

export type ManualTriggerConfigSchema = z.infer<
  typeof manualTriggerConfigSchema
>;

// ==================== Workflow Node & Edge ====================

export const workflowNodeSchema = z.object({
  id: z.string().min(1),
  type: NodeType,
  actionType: ActionType.optional(),
  label: z.string().min(1, "Node label is required"),
  config: z.record(z.any(), z.any()),
  position: z.object({
    x: z.number(),
    y: z.number(),
  }),
  data: z.record(z.any(), z.any()).optional(),
});

export type WorkflowNodeSchema = z.infer<typeof workflowNodeSchema>;

export const workflowEdgeSchema = z.object({
  id: z.string().min(1),
  source: z.string().min(1),
  target: z.string().min(1),
  sourceHandle: optionalStringSchema,
  targetHandle: optionalStringSchema,
  label: optionalStringSchema,
  condition: z.record(z.any(), z.any()).optional(),
});

export type WorkflowEdgeSchema = z.infer<typeof workflowEdgeSchema>;

// ==================== Workflow Definition ====================

export const workflowDefinitionSchema = z.object({
  nodes: z.array(workflowNodeSchema),
  edges: z.array(workflowEdgeSchema),
  viewport: z
    .object({
      x: z.number(),
      y: z.number(),
      zoom: z.number(),
    })
    .optional(),
});

export type WorkflowDefinitionSchema = z.infer<typeof workflowDefinitionSchema>;

// ==================== Workflow ====================

export const workflowSchema = z.object({
  id: optionalStringSchema,
  version: versionSchema,
  createdAt: timestampSchema,
  updatedAt: timestampSchema,
  organizationId: pulidSchema,
  businessUnitId: pulidSchema,

  name: z
    .string()
    .min(1, "Workflow name is required")
    .max(100, "Workflow name must be less than 100 characters"),
  description: optionalStringSchema,
  status: WorkflowStatus,
  triggerType: TriggerType,
  triggerConfig: z.record(z.any(), z.any()),
  currentVersionId: nullablePulidSchema,
  publishedVersionId: nullablePulidSchema,
  tags: z.array(z.string()).optional(),

  createdBy: pulidSchema,
  updatedBy: pulidSchema,
});

export type WorkflowSchema = z.infer<typeof workflowSchema>;

// ==================== Workflow Version ====================

export const workflowVersionSchema = z.object({
  id: optionalStringSchema,
  version: versionSchema,
  createdAt: timestampSchema,
  organizationId: pulidSchema,
  businessUnitId: pulidSchema,

  workflowId: pulidSchema,
  versionNumber: z.number().int().min(1),
  definition: workflowDefinitionSchema,
  isPublished: z.boolean(),
  publishedAt: timestampSchema,
  publishedBy: nullablePulidSchema,
  changelog: optionalStringSchema,

  createdBy: pulidSchema,
});

export type WorkflowVersionSchema = z.infer<typeof workflowVersionSchema>;

// ==================== Workflow Execution ====================

export const workflowExecutionSchema = z.object({
  id: optionalStringSchema,
  version: versionSchema,
  createdAt: timestampSchema,
  organizationId: pulidSchema,
  businessUnitId: pulidSchema,

  workflowId: pulidSchema,
  workflowVersionId: pulidSchema,
  status: ExecutionStatus,
  startedAt: timestampSchema,
  completedAt: timestampSchema,
  duration: z.number().int().min(0).optional(),
  triggerType: TriggerType,
  triggerData: z.record(z.any(), z.any()).optional(),
  inputData: z.record(z.any(), z.any()).optional(),
  outputData: z.record(z.any(), z.any()).optional(),
  errorMessage: optionalStringSchema,
  errorDetails: z.record(z.any(), z.any()).optional(),
  retryCount: z.number().int().min(0),
  maxRetries: z.number().int().min(0),
  temporalWorkflowId: optionalStringSchema,
  temporalRunId: optionalStringSchema,

  createdBy: pulidSchema,
});

export type WorkflowExecutionSchema = z.infer<typeof workflowExecutionSchema>;

// ==================== Workflow Execution Step ====================

export const workflowExecutionStepSchema = z.object({
  id: optionalStringSchema,
  createdAt: timestampSchema,

  executionId: pulidSchema,
  stepNumber: z.number().int().min(1),
  nodeId: pulidSchema,
  nodeType: NodeType,
  actionType: ActionType.optional(),
  status: ExecutionStatus,
  startedAt: timestampSchema,
  completedAt: timestampSchema,
  duration: z.number().int().min(0).optional(),
  inputData: z.record(z.any(), z.any()).optional(),
  outputData: z.record(z.any(), z.any()).optional(),
  errorMessage: optionalStringSchema,
  retryCount: z.number().int().min(0),
});

export type WorkflowExecutionStepSchema = z.infer<
  typeof workflowExecutionStepSchema
>;

// ==================== Workflow Template ====================

export const workflowTemplateSchema = z.object({
  id: optionalStringSchema,
  version: versionSchema,
  createdAt: timestampSchema,
  updatedAt: timestampSchema,
  organizationId: pulidSchema,
  businessUnitId: pulidSchema,

  name: z
    .string()
    .min(1, "Template name is required")
    .max(100, "Template name must be less than 100 characters"),
  description: optionalStringSchema,
  category: optionalStringSchema,
  definition: workflowDefinitionSchema,
  triggerType: TriggerType,
  triggerConfig: z.record(z.any(), z.any()),
  isSystem: z.boolean(),
  isPublic: z.boolean(),
  usageCount: z.number().int().min(0),
  tags: z.array(z.string()).optional(),

  createdBy: pulidSchema,
  updatedBy: pulidSchema,
});

export type WorkflowTemplateSchema = z.infer<typeof workflowTemplateSchema>;

// ==================== Request Schemas ====================

export const createWorkflowRequestSchema = z.object({
  name: z
    .string()
    .min(1, "Workflow name is required")
    .max(100, "Workflow name must be less than 100 characters"),
  description: optionalStringSchema,
  triggerType: TriggerType,
  triggerConfig: z.record(z.any(), z.any()),
  tags: z.array(z.string()).optional(),
});

export type CreateWorkflowRequestSchema = z.infer<
  typeof createWorkflowRequestSchema
>;

export const updateWorkflowRequestSchema = z.object({
  name: z
    .string()
    .min(1, "Workflow name is required")
    .max(100, "Workflow name must be less than 100 characters")
    .optional(),
  description: optionalStringSchema,
  triggerConfig: z.record(z.any(), z.any()).optional(),
  tags: z.array(z.string()).optional(),
});

export type UpdateWorkflowRequestSchema = z.infer<
  typeof updateWorkflowRequestSchema
>;

export const createVersionRequestSchema = z.object({
  changelog: optionalStringSchema,
});

export type CreateVersionRequestSchema = z.infer<
  typeof createVersionRequestSchema
>;

export const saveDefinitionRequestSchema = z.object({
  definition: workflowDefinitionSchema,
});

export type SaveDefinitionRequestSchema = z.infer<
  typeof saveDefinitionRequestSchema
>;

export const triggerWorkflowRequestSchema = z.object({
  triggerData: z.record(z.any(), z.any()).optional(),
});

export type TriggerWorkflowRequestSchema = z.infer<
  typeof triggerWorkflowRequestSchema
>;

export const createTemplateRequestSchema = z.object({
  name: z
    .string()
    .min(1, "Template name is required")
    .max(100, "Template name must be less than 100 characters"),
  description: optionalStringSchema,
  category: optionalStringSchema,
  definition: workflowDefinitionSchema,
  triggerType: TriggerType,
  triggerConfig: z.record(z.any(), z.any()),
  isPublic: z.boolean(),
  tags: z.array(z.string()).optional(),
});

export type CreateTemplateRequestSchema = z.infer<
  typeof createTemplateRequestSchema
>;

export const useTemplateRequestSchema = z.object({
  name: z
    .string()
    .min(1, "Workflow name is required")
    .max(100, "Workflow name must be less than 100 characters"),
  description: optionalStringSchema,
});

export type UseTemplateRequestSchema = z.infer<typeof useTemplateRequestSchema>;
