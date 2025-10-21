import * as z from "zod/v4";
import {
  optionalStringSchema,
  pulidSchema,
  timestampSchema,
  versionSchema,
} from "./helpers";
import { userSchema } from "./user-schema";

export const aiLogSchema = z.object({
  id: optionalStringSchema,
  version: versionSchema,
  timestamp: timestampSchema,
  organizationId: optionalStringSchema,
  businessUnitId: optionalStringSchema,

  userId: pulidSchema,
  prompt: z.string().min(1, { error: "Prompt is required" }),
  response: z.string().min(1, { error: "Response is required" }),
  model: z.string().min(1, { error: "Model is required" }),
  operation: z.string().min(1, { error: "Operation is required" }),
  object: z.string().min(1, { error: "Object is required" }),
  serviceTier: z.string().min(1, { error: "Service tier is required" }),
  promptTokens: z.number().int().positive(),
  completionTokens: z.number().int().positive(),
  totalTokens: z.number().int().positive(),
  reasoningTokens: z.number().int().positive(),
  user: userSchema.nullish(),
});

export type AILogSchema = z.infer<typeof aiLogSchema>;
