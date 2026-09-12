import { useT } from "@trenova/shared/i18n/use-t";
import { InfoPopover } from "@/components/info-popover";
import { KpiCard, KpiHeader, KpiSub } from "@/components/kpi/kpi-card";
import type { OshaLog } from "@/lib/graphql/worker-injury";
import { daysLost, LOG_COLUMNS } from "@/lib/osha-log";
import NumberFlow from "@number-flow/react";
import { CompositionBar } from "@trenova/shared/components/ui/composition-bar";
import { formatRate } from "@trenova/shared/lib/injury";
import { ActivityIcon, BedIcon, ClipboardListIcon, GaugeIcon } from "lucide-react";
import { useMemo } from "react";

const VALUE_CLASS = "font-mono text-[26px] leading-none font-semibold tracking-tight tabular-nums";

type OshaOverviewProps = {
  log: OshaLog;
};

/**
 * The year in four numbers: how many cases made the log, the two rates OSHA
 * and every insurer read, and the days those cases cost. Counted from the same
 * log the table shows, so the headline never disagrees with the rows.
 */
export function OshaOverview({ log }: OshaOverviewProps) {
  const t = useT();

  const { totals, summary } = log;
  const lost = useMemo(() => daysLost(totals), [totals]);
  const offTheLog = log.cases.length - totals.totalRecordableCases;
  const hours = summary?.totalHoursWorked ?? 0;
  const columnSegments = useMemo(
    () =>
      LOG_COLUMNS.map((column) => ({
        key: column.column,
        label: `${column.column} ${column.label}`,
        value: totals[column.key],
      })),
    [totals],
  );

  return (
    <div className="grid grid-cols-4 gap-3 lg:grid-cols-8">
      <KpiCard span={2}>
        <KpiHeader
          icon={<ClipboardListIcon className="size-[11px]" />}
          label={t("Recordable cases")}
          info={
            <InfoPopover title={t("Recordable cases")}>
              {t(
                "Cases that met the OSHA recording criteria this year, by the log column they landed in. A case kept on file but judged not recordable is counted below, not here.",
              )}
            </InfoPopover>
          }
        />
        <NumberFlow
          value={totals.totalRecordableCases}
          className={VALUE_CLASS}
          aria-label={t("Recordable cases")}
        />
        <CompositionBar
          size="sm"
          showLegend={false}
          className="mt-auto"
          aria-label={t("Cases by log column")}
          segments={columnSegments}
        />
        <KpiSub>{describeCases(log.cases.length, totals.openCases, offTheLog)}</KpiSub>
      </KpiCard>

      <KpiCard span={2}>
        <KpiHeader
          icon={<GaugeIcon className="size-[11px]" />}
          label={t("Incident rate")}
          info={
            <InfoPopover title={t("Incident rate")}>
              {t(
                "Recordable cases times 200,000, divided by the hours worked from the 300A figures: the rate per 100 full-time workers OSHA and insurers compare fleets on.",
              )}
            </InfoPopover>
          }
        />
        <span className={VALUE_CLASS} aria-label={t("Incident rate")}>
          {formatRate(log.totalRecordableIncidentRate)}
        </span>
        <KpiSub>
          {log.totalRecordableIncidentRate === null
            ? t("Needs the hours worked from the 300A figures")
            : t(
                "Recordable cases per 100 full-time workers, over {0} hours",
                hours.toLocaleString("en-US"),
              )}
        </KpiSub>
      </KpiCard>

      <KpiCard span={2}>
        <KpiHeader
          icon={<ActivityIcon className="size-[11px]" />}
          label={t("DART rate")}
          info={
            <InfoPopover title={t("DART rate")}>
              {t(
                "Cases with days away, restricted duty or job transfer, on the same 200,000-hour basis. It is the rate most workers' compensation carriers price on.",
              )}
            </InfoPopover>
          }
        />
        <span className={VALUE_CLASS} aria-label={t("DART rate")}>
          {formatRate(log.daysAwayRestrictedRate)}
        </span>
        <KpiSub>
          {log.daysAwayRestrictedRate === null
            ? t("Needs the hours worked from the 300A figures")
            : t("Cases with days away, restriction or transfer, per 100 full-time workers")}
        </KpiSub>
      </KpiCard>

      <KpiCard span={2}>
        <KpiHeader
          icon={<BedIcon className="size-[11px]" />}
          label={t("Days lost")}
          info={
            <InfoPopover title={t("Days lost")}>
              {t(
                "Calendar days away from work plus days on restricted duty or transfer, summed across the year's cases, each capped at 180 as the form requires.",
              )}
            </InfoPopover>
          }
        />
        <NumberFlow value={lost.total} className={VALUE_CLASS} aria-label={t("Days lost")} />
        <CompositionBar
          size="sm"
          showLegend={false}
          className="mt-auto"
          aria-label={t("Days lost by kind")}
          segments={[
            { key: "away", label: t("Away from work"), value: lost.away },
            { key: "restricted", label: t("Restricted or transferred"), value: lost.restricted },
          ]}
        />
        <KpiSub>
          {lost.total === 0
            ? t("No case kept anybody off their job")
            : t(
                "{0} away from work, {1} restricted or transferred",
                lost.away.toLocaleString("en-US"),
                lost.restricted.toLocaleString("en-US"),
              )}
        </KpiSub>
      </KpiCard>
    </div>
  );
}

function describeCases(recorded: number, openCases: number, offTheLog: number): string {
  if (recorded === 0) return "Nothing recorded this year";
  const parts: string[] = [];
  if (openCases > 0) {
    parts.push(`${openCases} still accruing days`);
  }
  if (offTheLog > 0) {
    parts.push(`${offTheLog} kept off the log`);
  }
  return parts.length > 0 ? parts.join(", ") : "Every case is closed";
}
