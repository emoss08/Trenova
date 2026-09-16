import type { InsightCategory, InsightSeverity } from "@/types/insight";
import type { InsightStatusFilter } from "./insight-filters";

/**
 * What each category is called in the interface.
 *
 * The enum values are the server's vocabulary; these are the reader's. Nobody
 * scanning their morning findings thinks in terms of "CostLeakage".
 */
export const CATEGORY_LABELS: Record<InsightCategory, string> = {
  ServiceQuality: "Service",
  CashFlow: "Cash",
  CostLeakage: "Cost",
  Compliance: "Compliance",
};

export const CATEGORY_DESCRIPTIONS: Record<InsightCategory, string> = {
  ServiceQuality: "On-time performance and service the customer feels",
  CashFlow: "Money earned but not yet collected",
  CostLeakage: "Money the operation is losing quietly",
  Compliance: "Exposure that takes capacity off the road",
};

export const SEVERITY_TONE: Record<InsightSeverity, "inactive" | "warning" | "outline"> = {
  Critical: "inactive",
  Warning: "warning",
  Info: "outline",
};

export const STATUS_LABELS: Record<InsightStatusFilter, string> = {
  active: "Open",
  dismissed: "Dismissed",
  closed: "Closed",
};

export const STATUS_DESCRIPTIONS: Record<InsightStatusFilter, string> = {
  active: "Findings that were present at the last refresh",
  dismissed: "Findings someone judged not worth acting on",
  closed: "Findings a later refresh no longer makes",
};
