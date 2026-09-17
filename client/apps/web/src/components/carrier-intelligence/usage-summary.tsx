import {
  spendCapProgress,
  summarizeUsageByDay,
  utcDayKeyToUnix,
  type SpendCapState,
} from "@/lib/carrier-intel-usage";
import {
  carrierIntelProviderLabel,
  formatOptionalDecimalCurrency,
} from "@/lib/carrier-intelligence";
import type { CarrierIntelUsageSummary } from "@/lib/graphql/carrier-intel-settings";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Progress } from "@trenova/shared/components/ui/progress";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@trenova/shared/components/ui/table";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { useMemo } from "react";

const progressVariantByState: Record<SpendCapState, "default" | "warning" | "error"> = {
  uncapped: "default",
  within: "default",
  soft: "warning",
  exceeded: "error",
};

export type CarrierIntelUsageSummaryViewProps = {
  usage: CarrierIntelUsageSummary;
  emptyMessage?: string;
};

export function CarrierIntelUsageSummaryView({
  usage,
  emptyMessage,
}: CarrierIntelUsageSummaryViewProps) {
  const t = useT();
  const progress = spendCapProgress(usage.monthToDate, usage.cap, usage.softCapPercent);
  const dailyRows = useMemo(() => summarizeUsageByDay(usage.daily), [usage.daily]);

  return (
    <div className="space-y-4">
      <div className="border-border space-y-2 rounded-md border p-3">
        <div className="flex flex-wrap items-baseline justify-between gap-2">
          <div className="flex items-baseline gap-2">
            <span className="text-2xl font-semibold" data-testid="usage-month-to-date">
              {formatOptionalDecimalCurrency(usage.monthToDate) ?? "-"}
            </span>
            <span className="text-muted-foreground text-sm">
              {progress.state === "uncapped"
                ? t("No monthly cap")
                : t("of {0}", formatOptionalDecimalCurrency(usage.cap) ?? "-")}
            </span>
          </div>
          <div className="flex items-center gap-2">
            <span className="text-muted-foreground text-xs">
              {t("Since {0}", formatUnixDateMedium(usage.monthStart, { timezone: "UTC" }))}
            </span>
            {progress.state === "exceeded" ? (
              <Badge variant="inactive">{t("Cap reached")}</Badge>
            ) : progress.state === "soft" ? (
              <Badge variant="warning">{t("Above {0}%", usage.softCapPercent)}</Badge>
            ) : null}
          </div>
        </div>
        {progress.state !== "uncapped" ? (
          <>
            <Progress value={progress.percent} variant={progressVariantByState[progress.state]} />
            <p className="text-muted-foreground text-xs">
              {t(
                "Administrators are warned at {0} ({1}%).",
                formatOptionalDecimalCurrency(progress.softCapAmount) ?? "-",
                usage.softCapPercent,
              )}
            </p>
          </>
        ) : null}
      </div>
      <div className="grid gap-4 lg:grid-cols-2">
        <div className="space-y-2">
          <h4 className="text-xs font-semibold uppercase">{t("By endpoint")}</h4>
          {usage.byEndpoint.length === 0 ? (
            <EmptyUsage message={emptyMessage} />
          ) : (
            <Table containerClassName="rounded-md border">
              <TableHeader>
                <TableRow>
                  <TableHead>{t("Endpoint")}</TableHead>
                  <TableHead className="text-right">{t("Calls")}</TableHead>
                  <TableHead className="text-right">{t("Billable")}</TableHead>
                  <TableHead className="text-right">{t("Cost")}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {usage.byEndpoint.map((row) => (
                  <TableRow key={`${row.provider}-${row.endpoint}`}>
                    <TableCell>
                      <div className="flex flex-col">
                        <span className="text-sm">{row.endpoint}</span>
                        <span className="text-muted-foreground text-2xs">
                          {carrierIntelProviderLabel(row.provider)}
                        </span>
                      </div>
                    </TableCell>
                    <TableCell className="text-right">{row.calls.toLocaleString()}</TableCell>
                    <TableCell className="text-right">
                      {row.billableUnits.toLocaleString()}
                    </TableCell>
                    <TableCell className="text-right">
                      {formatOptionalDecimalCurrency(row.estimatedCost) ?? "-"}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </div>
        <div className="space-y-2">
          <h4 className="text-xs font-semibold uppercase">{t("Daily")}</h4>
          {dailyRows.length === 0 ? (
            <EmptyUsage message={emptyMessage} />
          ) : (
            <Table containerClassName="max-h-72 rounded-md border">
              <TableHeader>
                <TableRow>
                  <TableHead>{t("Day")}</TableHead>
                  <TableHead className="text-right">{t("Calls")}</TableHead>
                  <TableHead className="text-right">{t("Billable")}</TableHead>
                  <TableHead className="text-right">{t("Cost")}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {dailyRows.map((row) => (
                  <TableRow key={row.day}>
                    <TableCell>
                      {formatUnixDateMedium(utcDayKeyToUnix(row.day), { timezone: "UTC" })}
                    </TableCell>
                    <TableCell className="text-right">{row.calls.toLocaleString()}</TableCell>
                    <TableCell className="text-right">
                      {row.billableUnits.toLocaleString()}
                    </TableCell>
                    <TableCell className="text-right">
                      {formatOptionalDecimalCurrency(row.estimatedCost) ?? "-"}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </div>
      </div>
    </div>
  );
}

function EmptyUsage({ message }: { message?: string }) {
  const t = useT();

  return (
    <div className="border-border bg-muted/20 text-muted-foreground rounded-md border p-3 text-sm">
      {message ?? t("No provider calls this month.")}
    </div>
  );
}
