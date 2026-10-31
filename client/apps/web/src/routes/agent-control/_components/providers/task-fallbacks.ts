import type { AITask } from "@/types/ai-provider";
import { defineLabels } from "@trenova/shared/i18n/labels";

/** What Trenova does about a task while no provider takes it. */
export const TASK_FALLBACKS: Record<AITask, string> = defineLabels({
  AssistantChat: "The assistant can't answer",
  BillingDiagnosis: "Billing holds wait for a person",
  DocumentExtraction: "Documents wait for manual entry",
  DocumentClassification: "Documents are filed by hand",
  ScopeClassification: "Built-in rules check scope",
  OperationalInsights: "Insights show without narration",
  DailyBriefing: "No morning briefing",
  FormulaAssistant: "The Desk's model helps with formulas",
  QueryCompose: "Tables are filtered by hand",
  ChapterTitling: "Chapters are named by the Desk's own model",
  InboundClassification: "Inbound mail is sorted by hand",
  AccountingMapping: "Accounts are mapped by hand",
  RemittanceExtraction: "PDF remittances are refused; upload the CSV",
  PaperworkReading: "Paperwork is read once its conversion finishes",
  PaletteAnswer: "The Desk's model answers palette questions",
  EvaluationJudge: "Evaluations go unscored",
  Embedding: "Search uses words only",
  General: "Unrouted work is refused",
});
