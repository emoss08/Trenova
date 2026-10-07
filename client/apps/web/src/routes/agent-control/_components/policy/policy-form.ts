import type { AgentControl } from "@/lib/graphql/agent-control";
import type { AgentControlInput } from "@trenova/graphql/generated/graphql";
import { z } from "zod";

/** The organization-wide settings as the policy editor holds them. */
export const policyFormSchema = z.object({
  earnedAutonomy: z.boolean(),
  promotionThreshold: z.number().int().min(1).max(1000),
  learnsFromWork: z.boolean(),
  personMonthlyMessages: z.number().int().min(0),
  aiTrainingConsent: z.boolean(),
  version: z.number(),
});

export type PolicyFormValues = z.infer<typeof policyFormSchema>;

export function toPolicyForm(control: AgentControl): PolicyFormValues {
  return {
    earnedAutonomy: control.earnedAutonomy,
    promotionThreshold: control.promotionThreshold,
    learnsFromWork: !control.learningOff,
    personMonthlyMessages: control.personMonthlyMessages,
    aiTrainingConsent: control.aiTrainingConsent,
    version: control.version,
  };
}

/**
 * The update the editor sends. The pause switch is not the editor's and is
 * sent as it stands; training consent only when it changed, so saving anything
 * else never re-records who consented.
 */
export function toControlInput(
  values: PolicyFormValues,
  loaded: PolicyFormValues,
  shadowMode: boolean,
): AgentControlInput {
  return {
    shadowMode,
    earnedAutonomy: values.earnedAutonomy,
    promotionThreshold: values.promotionThreshold,
    learningOff: !values.learnsFromWork,
    personMonthlyMessages: values.personMonthlyMessages,
    version: values.version,
    ...(values.aiTrainingConsent === loaded.aiTrainingConsent
      ? {}
      : { aiTrainingConsent: values.aiTrainingConsent }),
  };
}

/**
 * Whether saving makes tools that already earned it move up: earned autonomy
 * turned on, or its threshold lowered while on.
 */
export function promotesOnSave(values: PolicyFormValues, loaded: PolicyFormValues): boolean {
  if (!values.earnedAutonomy) {
    return false;
  }
  return !loaded.earnedAutonomy || values.promotionThreshold < loaded.promotionThreshold;
}
