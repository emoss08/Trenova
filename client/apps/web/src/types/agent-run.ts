import { z } from "zod";

export const agentRunStatusSchema = z.enum([
  "Pending",
  "Running",
  "AwaitingDecision",
  "Completed",
  "ShadowCompleted",
  "Failed",
]);

export const agentRunSchema = z.object({
  id: z.string(),
  agentDefinitionId: z.string().nullish(),
  agentType: z.string(),
  subjectType: z.string(),
  subjectId: z.string(),
  status: agentRunStatusSchema,
  trigger: z.string(),
  summary: z.string().nullish(),
  workflowId: z.string().nullish(),
  createdAt: z.number(),
});

export const startAgentRunRequestSchema = z.object({
  agentDefinitionId: z.string().optional(),
  systemKey: z.string().optional(),
  subjectType: z.string().optional(),
  subjectId: z.string().optional(),
});

export type AgentRun = z.infer<typeof agentRunSchema>;
export type StartAgentRunRequest = z.infer<typeof startAgentRunRequestSchema>;
