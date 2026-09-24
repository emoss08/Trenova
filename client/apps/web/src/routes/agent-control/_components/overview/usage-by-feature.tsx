import { SectionPanel } from "@/components/section-panel";
import { aiUsageFeatureLabel, formatLatency, formatTokens, formatUsd } from "@/lib/ai-usage-format";
import type { AIUsageSummary } from "@/lib/graphql/ai-usage";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@trenova/shared/components/ui/table";
import { useT } from "@trenova/shared/i18n/use-t";

export type UsageFeatureSlice = AIUsageSummary["byFeature"][number];

/**
 * Where the window's model calls went: which feature made them, what they
 * consumed and cost, and how long a person waited. Spend on an unpriced
 * provider shows as a dash rather than as free.
 */
export function UsageByFeature({
  slices,
  days,
}: {
  slices: readonly UsageFeatureSlice[];
  days: number;
}) {
  const t = useT();

  if (slices.length === 0) {
    return null;
  }

  return (
    <SectionPanel
      title={t("Usage by feature")}
      count={slices.length}
      help={t("Model calls over the last {0} days, by the feature that made them.", days)}
    >
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>{t("Feature")}</TableHead>
            <TableHead className="text-right">{t("Calls")}</TableHead>
            <TableHead className="text-right">{t("Tokens")}</TableHead>
            <TableHead className="text-right">{t("Spend")}</TableHead>
            <TableHead className="text-right">{t("Median response")}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {slices.map((slice) => (
            <TableRow key={slice.feature ?? "other"}>
              <TableCell>{aiUsageFeatureLabel(slice.feature, t)}</TableCell>
              <TableCell className="text-right tabular-nums">
                {slice.calls.toLocaleString()}
                {slice.failed > 0 && (
                  <span className="text-muted-foreground">
                    {" · "}
                    {t("{0} failed", slice.failed)}
                  </span>
                )}
              </TableCell>
              <TableCell className="text-right tabular-nums">
                {formatTokens(slice.inputTokens + slice.outputTokens)}
              </TableCell>
              <TableCell className="text-right tabular-nums">
                {slice.pricedCalls > 0 ? (formatUsd(slice.costUsd) ?? "—") : "—"}
                {slice.pricedCalls > 0 && slice.pricedCalls < slice.calls && (
                  <span className="text-muted-foreground">
                    {" · "}
                    {t("Partial")}
                  </span>
                )}
              </TableCell>
              <TableCell className="text-right tabular-nums">
                {slice.latencyP50Ms > 0 ? formatLatency(slice.latencyP50Ms) : "—"}
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </SectionPanel>
  );
}
