import type { AgentTemplate } from "@/types/assistant";
import type { AgentFormValues } from "./agent-form-schema";

export type TemplatePatch = Partial<
  Pick<
    AgentFormValues,
    | "template"
    | "instructions"
    | "toolNames"
    | "triggerMode"
    | "cronExpression"
    | "eventKinds"
    | "autonomyCeiling"
    | "dataAccessCeiling"
    | "dailyRunLimit"
    | "outputMode"
    | "contextProviders"
    | "shadowMode"
  >
>;

/**
 * What picking a template changes: only fields the person has not touched.
 * The ceiling, data access and output mode are set outright, since a
 * template's whole point is to suggest how much it should do, what it should
 * see and what it should produce. A daily run cap fills only an agent with
 * none, so a cap the person chose is never loosened. A starter that asks for
 * shadow mode turns it on; picking another template never turns it off.
 */
export function applyTemplateStarter(
  current: AgentFormValues,
  template: AgentTemplate | null,
): TemplatePatch {
  if (!template) {
    return { template: null };
  }

  const patch: TemplatePatch = { template: template.template };

  if (current.instructions.trim() === "" && template.starterInstructions !== "") {
    patch.instructions = template.starterInstructions;
  }
  if (current.toolNames.length === 0 && template.starterTools.length > 0) {
    patch.toolNames = [...template.starterTools];
  }
  if (current.triggerMode === "Chat" && template.starterTrigger !== "Chat") {
    patch.triggerMode = template.starterTrigger;
    if (template.starterTrigger === "Scheduled" && template.starterCron !== "") {
      patch.cronExpression = template.starterCron;
    }
    if (template.starterTrigger === "Event" && template.starterEvents.length > 0) {
      patch.eventKinds = [...template.starterEvents];
    }
  }
  patch.autonomyCeiling = template.starterCeiling;
  patch.dataAccessCeiling = template.starterDataAccess;
  patch.outputMode = template.starterOutput;
  if (current.dailyRunLimit === 0 && template.starterDailyRunLimit > 0) {
    patch.dailyRunLimit = template.starterDailyRunLimit;
  }
  if (template.starterShadow && !current.shadowMode) {
    patch.shadowMode = true;
  }
  if (current.contextProviders.length === 0 && template.contextProviders.length > 0) {
    patch.contextProviders = [...template.contextProviders];
  }

  return patch;
}
