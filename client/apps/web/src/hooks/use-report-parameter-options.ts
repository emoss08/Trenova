import { useReportCatalog } from "@/hooks/use-reports";
import {
  hasAllowedValues,
  reportParameterOptions,
  type ReportParameterOptions,
} from "@/lib/report-parameter-options";
import type { ReportIR } from "@/types/report";

/**
 * The options of a report's parameters, named from the catalog. The catalog is read only when
 * `enabled` and the report has a parameter with allowed values; until it arrives (or when it
 * cannot be read) every option reads its code in words.
 */
export function useReportParameterOptions(
  ir: ReportIR | null | undefined,
  enabled = true,
): ReportParameterOptions {
  const catalog = useReportCatalog(enabled && hasAllowedValues(ir?.parameters));
  return reportParameterOptions(ir, catalog.data);
}
