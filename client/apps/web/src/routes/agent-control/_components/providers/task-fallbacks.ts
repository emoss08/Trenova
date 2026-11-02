import type { AITask } from "@/types/ai-provider";
import { defineLabels } from "@trenova/shared/i18n/labels";

/**
 * What Trenova does about a task while no provider takes it, for the tasks the catalog
 * routes. Nova's tasks are absent: they run on the provider chosen for Nova.
 */
export const TASK_FALLBACKS: Partial<Record<AITask, string>> = defineLabels({
  AssistantChat: "The assistant can't answer",
  ScopeClassification: "Built-in rules check scope",
  EvaluationJudge: "Evaluations go unscored",
  Embedding: "Search uses words only",
  General: "Unrouted work is refused",
});
