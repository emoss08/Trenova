import { z } from "zod";
import { createLimitOffsetResponse } from "./server";

export const emailProviderSchema = z.enum(["Resend", "Postmark"]);
export const emailProfileStatusSchema = z.enum(["Active", "Inactive"]);
export const emailPurposeSchema = z.enum([
  "General",
  "Billing",
  "Reporting",
  "Operations",
  "Authentication",
  "Notifications",
]);
export const emailMessageStatusSchema = z.enum([
  "Queued",
  "Sending",
  "Sent",
  "Delivered",
  "Failed",
  "Bounced",
  "Complained",
  "Opened",
  "Clicked",
  "Suppressed",
]);
export const emailSuppressionReasonSchema = z.enum([
  "HardBounce",
  "Complaint",
  "SoftBounceLimit",
  "Manual",
]);

export const emailProfileSchema = z.object({
  id: z.string().optional(),
  businessUnitId: z.string().optional(),
  organizationId: z.string().optional(),
  name: z.string().min(1),
  description: z.string().optional().nullable(),
  senderName: z.string().min(1),
  senderEmail: z.string().email(),
  replyToEmail: z.string().email().optional().or(z.literal("")),
  provider: emailProviderSchema.default("Resend"),
  status: emailProfileStatusSchema.default("Active"),
  version: z.number().optional(),
  createdAt: z.number().optional(),
  updatedAt: z.number().optional(),
});

export const emailProfileAssignmentSchema = z.object({
  id: z.string().optional(),
  purpose: emailPurposeSchema,
  profileId: z.string(),
  profile: emailProfileSchema.optional(),
});

export const emailMessageSchema = z.object({
  id: z.string(),
  profileId: z.string(),
  purpose: emailPurposeSchema,
  provider: emailProviderSchema,
  idempotencyKey: z.string(),
  providerMessageId: z.string().optional(),
  status: emailMessageStatusSchema,
  attempts: z.number(),
  fromEmail: z.string(),
  fromName: z.string(),
  toRecipients: z.array(z.string()),
  ccRecipients: z.array(z.string()).optional().nullable(),
  bccRecipients: z.array(z.string()).optional().nullable(),
  subject: z.string(),
  lastError: z.string().optional(),
  sentAt: z.number().optional(),
  deliveredAt: z.number().optional(),
  failedAt: z.number().optional(),
  createdAt: z.number(),
  updatedAt: z.number(),
  profile: emailProfileSchema.optional(),
});

export const emailSuppressionSchema = z.object({
  id: z.string().optional(),
  emailAddress: z.string().email(),
  reason: emailSuppressionReasonSchema,
  provider: emailProviderSchema.optional().nullable(),
  sourceEventId: z.string().optional().nullable(),
  notes: z.string().optional().nullable(),
  createdAt: z.number().optional(),
});

export const testEmailProfileRequestSchema = z.object({
  to: z.string().email(),
  subject: z.string().min(1),
  html: z.string().optional().default(""),
  text: z.string().optional().default(""),
});

export const emailProfileListSchema = createLimitOffsetResponse(emailProfileSchema);
export const emailMessageListSchema = createLimitOffsetResponse(emailMessageSchema);
export const emailSuppressionListSchema = createLimitOffsetResponse(emailSuppressionSchema);

export type EmailProfile = z.infer<typeof emailProfileSchema>;
export type EmailProfileAssignment = z.infer<typeof emailProfileAssignmentSchema>;
export type EmailMessage = z.infer<typeof emailMessageSchema>;
export type EmailSuppression = z.infer<typeof emailSuppressionSchema>;
export type TestEmailProfileRequest = z.infer<typeof testEmailProfileRequestSchema>;
