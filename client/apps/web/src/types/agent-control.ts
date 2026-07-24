import { z } from "zod";

const HOUR_IN_SECONDS = 3600;
const MIN_DECISION_TIMEOUT = 5 * 60;
const MAX_DECISION_TIMEOUT = 7 * 24 * HOUR_IN_SECONDS;

export const agentControlSchema = z.object({
  billingAgentEnabled: z.boolean(),
  shadowMode: z.boolean(),
  decisionTimeoutSeconds: z
    .number({ message: "Decision timeout is required" })
    .int("Decision timeout must be a whole number of seconds")
    .min(MIN_DECISION_TIMEOUT, "Decision timeout must be at least 5 minutes")
    .max(MAX_DECISION_TIMEOUT, "Decision timeout cannot exceed 7 days"),
});

export type AgentControlFormValues = z.infer<typeof agentControlSchema>;
