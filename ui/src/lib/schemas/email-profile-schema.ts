/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import { Status } from "@/types/common";
import { z } from "zod/v4";
import {
  nullableBigIntegerSchema,
  nullableIntegerSchema,
  nullableStringSchema,
  optionalStringSchema,
  timestampSchema,
  versionSchema,
} from "./helpers";

export const ProviderType = z.enum([
  "SMTP",
  "SendGrid",
  "AWS_SES",
  "Mailgun",
  "Postmark",
  "Exchange",
  "Office365",
]);

export const AuthType = z.enum([
  "Plain",
  "Login",
  "CRAMMD5",
  "OAuth2",
  "APIKey",
]);

export const EncryptionType = z.enum(["None", "SSL_TLS", "StartTLS"]);

export const TemplateCategory = z.enum([
  "Notification",
  "Marketing",
  "System",
  "Custom",
]);

export const QueueStatus = z.enum([
  "Pending",
  "Processing",
  "Sent",
  "Failed",
  "Scheduled",
  "Cancelled",
]);

export const Priority = z.enum(["High", "Medium", "Low"]);

export const LogStatus = z.enum([
  "Delivered",
  "Opened",
  "Clicked",
  "Bounced",
  "Complained",
  "Unsubscribed",
  "Rejected",
]);

export const BounceType = z.enum(["Hard", "Soft", "Block"]);

export const emailProfileSchema = z.object({
  id: optionalStringSchema,
  version: versionSchema,
  createdAt: timestampSchema,
  updatedAt: timestampSchema,
  organizationId: optionalStringSchema,
  businessUnitId: optionalStringSchema,

  name: z.string().min(1, { error: "Name is required" }),
  description: z.string().optional(),
  status: z.enum(Status),
  providerType: ProviderType,
  authType: AuthType,
  encryptionType: EncryptionType,
  host: nullableStringSchema,
  username: nullableStringSchema,
  password: nullableStringSchema,
  apiKey: nullableStringSchema,
  oauth2ClientId: nullableStringSchema,
  oauth2TenantId: nullableStringSchema,
  fromAddress: nullableStringSchema,
  fromName: nullableStringSchema,
  replyTo: nullableStringSchema,
  port: nullableIntegerSchema,
  maxConnections: z
    .number()
    .min(1, { error: "Max Connections is required" })
    .default(5),
  timeoutSeconds: z
    .number()
    .min(1, { error: "Timeout Seconds is required" })
    .default(30),
  retryCount: z
    .number()
    .min(1, { error: "Retry Count is required" })
    .default(3),
  rateLimitPerMinute: z
    .number()
    .min(1, { error: "Rate Limit Per Minute is required" })
    .default(60),
  retryDelaySeconds: z
    .number()
    .min(1, { error: "Retry Delay Seconds is required" })
    .default(5),
  rateLimitPerHour: z
    .number()
    .min(1, { error: "Rate Limit Per Hour is required" })
    .default(1000),
  rateLimitPerDay: z
    .number()
    .min(1, { error: "Rate Limit Per Day is required" })
    .default(10000),
  isDefault: z.boolean().default(false),
  metadata: z.record(z.string(), z.any()).nullish(),
});

export const templateSchema = z.object({
  id: optionalStringSchema,
  version: versionSchema,
  createdAt: timestampSchema,
  updatedAt: timestampSchema,
  organizationId: optionalStringSchema,
  businessUnitId: optionalStringSchema,

  name: z.string().min(1, { error: "Name is required" }),
  slug: z.string().min(1, { error: "Slug is required" }),
  description: z.string().optional(),
  category: TemplateCategory,
  isSystem: z.boolean().default(false),
  isActive: z.boolean().default(true),
  status: z.enum(Status),
  subjectTemplate: z.string().min(1, { error: "Subject Template is required" }),
  htmlTemplate: z.string().min(1, { error: "HTML Template is required" }),
  textTemplate: z.string().optional(),
  variablesSchema: z.record(z.string(), z.any()).optional(),
  metadata: z.record(z.string(), z.any()).optional(),
});

export const queueSchema = z.object({
  id: optionalStringSchema,
  version: versionSchema,
  createdAt: timestampSchema,
  updatedAt: timestampSchema,
  organizationId: optionalStringSchema,
  businessUnitId: optionalStringSchema,

  // * Core Fields
  templateId: nullableStringSchema,
  toAddresses: z
    .array(z.string())
    .min(1, { error: "To Addresses is required" }),
  ccAddresses: z.array(z.string()).optional(),
  bccAddresses: z.array(z.string()).optional(),
  subject: z.string().min(1, { error: "Subject is required" }),
  htmlBody: z.string().min(1, { error: "HTML Body is required" }),
  textBody: z.string().optional(),
  attachments: z.array(z.any()).optional(),
  priority: Priority,
  status: QueueStatus,
  scheduledAt: nullableBigIntegerSchema,
  sentAt: nullableBigIntegerSchema,
  errorMessage: z.string().optional(),
  retryCount: z
    .number()
    .min(0, { error: "Retry Count must be greater than 0" })
    .default(0),
  templateVariables: z.record(z.string(), z.any()).optional(),
  metadata: z.record(z.string(), z.any()).optional(),

  // * Relationships
  profile: emailProfileSchema.nullish(),
  template: templateSchema.nullish(),
});

export const logSchema = z.object({
  id: optionalStringSchema,
  version: versionSchema,
  createdAt: timestampSchema,
  updatedAt: timestampSchema,
  organizationId: optionalStringSchema,
  businessUnitId: optionalStringSchema,

  queueId: z.string().min(1, { error: "Queue ID is required" }),
  messageId: z.string().min(1, { error: "Message ID is required" }),
  status: LogStatus,
  providerResponse: z.string().optional(),
  openedAt: nullableBigIntegerSchema,
  clickedAt: nullableBigIntegerSchema,
  bouncedAt: nullableBigIntegerSchema,
  complainedAt: nullableBigIntegerSchema,
  unsubscribedAt: nullableBigIntegerSchema,
  bounceType: BounceType,
  bounceReason: z.string().optional(),
  webhookEvents: z.array(z.any()).optional(),
  userAgent: z.string().optional(),
  ipAddress: z.string().optional(),
  clickedUrls: z.array(z.string()).optional(),
  metadata: z.record(z.string(), z.any()).optional(),

  // * Relationships
  queue: queueSchema.nullish(),
});

export type EmailProfileSchema = z.infer<typeof emailProfileSchema>;

export type TemplateSchema = z.infer<typeof templateSchema>;

export type QueueSchema = z.infer<typeof queueSchema>;

export type LogSchema = z.infer<typeof logSchema>;
