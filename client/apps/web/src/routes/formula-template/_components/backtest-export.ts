import { buildCsv, type ExportColumn } from "@/lib/data-table-export";
import type {
  BacktestResult,
  FormulaTemplateVersion,
} from "@trenova/shared/types/formula-template";

const BACKTEST_COLUMNS: ExportColumn<BacktestResult>[] = [
  { id: "shipmentId", header: "Shipment ID", getValue: (row) => row.shipmentId },
  { id: "proNumber", header: "Pro #", getValue: (row) => row.proNumber },
  { id: "currentAmount", header: "Current", getValue: (row) => row.currentAmount },
  { id: "candidateAmount", header: "Candidate", getValue: (row) => row.candidateAmount },
  { id: "delta", header: "Delta", getValue: (row) => row.delta },
  { id: "deltaPct", header: "Delta %", getValue: (row) => row.deltaPct },
  {
    id: "guardrailApplied",
    header: "Guardrail Clamped",
    getValue: (row) => (row.guardrailApplied ? "Yes" : "No"),
  },
  { id: "currentError", header: "Current Error", getValue: (row) => row.currentError ?? "" },
  {
    id: "candidateError",
    header: "Candidate Error",
    getValue: (row) => row.candidateError ?? "",
  },
];

export function backtestCsv(results: BacktestResult[]): string {
  return buildCsv(results, BACKTEST_COLUMNS);
}

export function describeVersionOption(
  version: FormulaTemplateVersion,
  formatDate: (unixSeconds: number) => string,
): string {
  const tags = version.tags ?? [];
  return [
    `v${version.versionNumber}`,
    tags.length > 0 ? tags.join(", ") : null,
    formatDate(version.createdAt),
  ]
    .filter(Boolean)
    .join(" · ");
}
