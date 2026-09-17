import {
  INTEL_EMPTY_VALUE,
  probabilityToPercent,
  sortBasicMeasures,
} from "@/lib/carrier-intelligence";
import type { CarrierIntelProfile } from "@/lib/graphql/carrier-intelligence";
import { formatNumber } from "@trenova/shared/i18n/format";
import { useT } from "@trenova/shared/i18n/use-t";
import { csaBasicHint, csaBasicLabel } from "@trenova/shared/lib/csa";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { formatPercent } from "@trenova/shared/lib/utils";
import {
  EmptyValue,
  Fact,
  FactGrid,
  MiniTable,
  Muted,
  SectionNote,
  dateOrDash,
  numberOrDash,
  textOrDash,
} from "../intel-facts";
import { OutOfServiceRate } from "../carrier-key-facts";
import { StatusDot, type StatusTone } from "../status-dot";
import { ProfileTab, type ProfileSectionProps } from "./profile-tab";

type BasicMeasure = NonNullable<CarrierIntelProfile["basics"]>[number];

function BasicName({ measure }: { measure: BasicMeasure }) {
  const t = useT();
  const flagged = measure.alert || measure.roadsideAlert;
  return (
    <span className="inline-flex items-center gap-2" title={csaBasicHint(measure.basic)}>
      {flagged ? <StatusDot tone="high" /> : null}
      <span>{csaBasicLabel(measure.basic)}</span>
      {flagged ? (
        <span className="sr-only">{measure.roadsideAlert ? t("Roadside alert") : t("Alert")}</span>
      ) : null}
      {measure.acIndicator ? <Muted>{t("acute/critical")}</Muted> : null}
    </span>
  );
}

function BasicsTable({ basics }: { basics: readonly BasicMeasure[] }) {
  const t = useT();
  const showPercentile = basics.some((measure) => measure.percentile !== null);
  const measuredAt = basics.reduce<number | null>(
    (latest, measure) =>
      measure.measuredAt !== null && (latest === null || measure.measuredAt > latest)
        ? measure.measuredAt
        : latest,
    null,
  );
  const alerts = basics.filter((measure) => measure.alert || measure.roadsideAlert).length;

  return (
    <div className="flex flex-col gap-2" data-testid="csa-basics">
      <MiniTable
        columns={[
          { id: "basic", label: t("BASIC") },
          { id: "measure", label: t("Measure"), align: "right" },
          { id: "violations", label: t("Violations"), align: "right" },
          { id: "oos", label: t("OOS violations"), align: "right" },
          ...(showPercentile
            ? [{ id: "percentile", label: t("Percentile"), align: "right" as const }]
            : []),
        ]}
        rows={basics.map((measure) => ({
          id: measure.basic,
          cells: [
            <BasicName key="basic" measure={measure} />,
            numberOrDash(measure.measure, { maximumFractionDigits: 2 }),
            numberOrDash(measure.violations),
            numberOrDash(measure.oosViolations),
            ...(showPercentile
              ? [
                  measure.percentile !== null ? (
                    formatPercent(measure.percentile, 0)
                  ) : (
                    <EmptyValue key="percentile" />
                  ),
                ]
              : []),
          ],
        }))}
      />
      <SectionNote>
        {[
          alerts > 0
            ? t("{0, plural, one {# BASIC over threshold} other {# BASICs over threshold}}", alerts)
            : t("No BASIC over threshold"),
          measuredAt ? t("measured {0}", formatUnixDateMedium(measuredAt)) : null,
        ]
          .filter(Boolean)
          .join(" · ")}
      </SectionNote>
    </div>
  );
}

function benchmarkTone(value: boolean | null): StatusTone {
  return value === true ? "medium" : value === false ? "success" : "neutral";
}

export function SafetySection({ profile, provider }: ProfileSectionProps) {
  const t = useT();
  const { safety, inspections, crashes, benchmarks } = profile;
  const basics = sortBasicMeasures(profile.basics ?? []);

  return (
    <ProfileTab
      profile={profile}
      provider={provider}
      parts={[
        {
          section: "Safety",
          title: t("Rating"),
          hasData: safety !== null,
          render: () =>
            safety ? (
              <FactGrid>
                <Fact label={t("Safety rating")}>
                  {safety.rating ?? <Muted>{t("Not rated")}</Muted>}
                </Fact>
                <Fact label={t("Rating date")}>{dateOrDash(safety.ratingDate)}</Fact>
                <Fact label={t("Out-of-service order")}>
                  {safety.outOfServiceOrder === null ? (
                    <EmptyValue />
                  ) : safety.outOfServiceOrder ? (
                    <span className="inline-flex items-center gap-2">
                      <StatusDot tone="critical" />
                      {safety.outOfServiceAt
                        ? t("Since {0}", formatUnixDateMedium(safety.outOfServiceAt))
                        : t("In effect")}
                    </span>
                  ) : (
                    t("None")
                  )}
                </Fact>
                <Fact label={t("Inspection selection (ISS)")}>
                  {safety.issValue !== null ? (
                    <span>
                      {formatNumber(safety.issValue)}
                      {safety.issRecommendation ? (
                        <Muted> · {safety.issRecommendation}</Muted>
                      ) : null}
                    </span>
                  ) : (
                    textOrDash(safety.issRecommendation)
                  )}
                </Fact>
                <Fact label={t("Risk score")}>
                  {safety.riskScore ? (
                    <span>
                      {safety.riskScore}
                      {safety.riskProbability !== null ? (
                        <Muted>
                          {" "}
                          · {formatPercent(probabilityToPercent(safety.riskProbability))}
                        </Muted>
                      ) : null}
                    </span>
                  ) : (
                    <EmptyValue />
                  )}
                </Fact>
                <Fact label={t("Safety score")}>
                  {numberOrDash(safety.safetyScore, { maximumFractionDigits: 1 })}
                </Fact>
                <Fact label={t("Latest review")}>
                  {safety.latestReviewType ? (
                    <span>
                      {safety.latestReviewType}
                      {safety.latestReviewAt ? (
                        <Muted> · {formatUnixDateMedium(safety.latestReviewAt)}</Muted>
                      ) : null}
                    </span>
                  ) : (
                    <EmptyValue />
                  )}
                </Fact>
              </FactGrid>
            ) : null,
        },
        {
          section: "Basics",
          title: t("CSA BASICs"),
          hasData: basics.length > 0,
          render: () => <BasicsTable basics={basics} />,
        },
        {
          section: "Inspections",
          title: t("Inspections"),
          hasData: inspections !== null,
          render: () =>
            inspections ? (
              <div className="flex flex-col gap-2">
                <MiniTable
                  columns={[
                    { id: "type", label: t("Type") },
                    { id: "count", label: t("Inspections"), align: "right" },
                    { id: "oos", label: t("Out of service"), align: "right" },
                    { id: "rate", label: t("OOS rate"), align: "right" },
                  ]}
                  rows={[
                    {
                      id: "driver",
                      label: t("Driver"),
                      count: inspections.driver,
                      oos: inspections.driverOos,
                      rate: inspections.driverOosRate,
                      national: inspections.nationalDriverOosRate,
                    },
                    {
                      id: "vehicle",
                      label: t("Vehicle"),
                      count: inspections.vehicle,
                      oos: inspections.vehicleOos,
                      rate: inspections.vehicleOosRate,
                      national: inspections.nationalVehicleOosRate,
                    },
                    {
                      id: "hazmat",
                      label: t("Hazmat"),
                      count: inspections.hazmat,
                      oos: inspections.hazmatOos,
                      rate: inspections.hazmatOosRate,
                      national: inspections.nationalHazmatOosRate,
                    },
                  ].map((row) => ({
                    id: row.id,
                    cells: [
                      row.label,
                      numberOrDash(row.count),
                      numberOrDash(row.oos),
                      <OutOfServiceRate key="rate" rate={row.rate} national={row.national} />,
                    ],
                  }))}
                />
                <SectionNote>
                  {t(
                    "{0} inspections · last on {1}",
                    inspections.total !== null
                      ? formatNumber(inspections.total)
                      : INTEL_EMPTY_VALUE,
                    inspections.lastInspectionAt
                      ? formatUnixDateMedium(inspections.lastInspectionAt)
                      : INTEL_EMPTY_VALUE,
                  )}
                </SectionNote>
              </div>
            ) : null,
        },
        {
          section: "Crashes",
          title: t("Crashes"),
          hasData: crashes !== null,
          render: () =>
            crashes ? (
              <FactGrid className="grid-cols-2 sm:grid-cols-5">
                <Fact label={t("Total")}>{numberOrDash(crashes.total)}</Fact>
                <Fact label={t("Fatal")}>
                  <span className="inline-flex items-center gap-2">
                    {(crashes.fatal ?? 0) > 0 ? <StatusDot tone="critical" /> : null}
                    {numberOrDash(crashes.fatal)}
                  </span>
                </Fact>
                <Fact label={t("Injury")}>{numberOrDash(crashes.injury)}</Fact>
                <Fact label={t("Tow-away")}>{numberOrDash(crashes.tow)}</Fact>
                <Fact label={t("Last crash")}>{dateOrDash(crashes.lastCrashAt)}</Fact>
              </FactGrid>
            ) : null,
        },
        {
          section: "Benchmarks",
          title: t("Benchmarks"),
          hasData: benchmarks !== null,
          render: () =>
            benchmarks ? (
              <ul className="divide-border divide-y">
                {[
                  {
                    id: "inspections",
                    label: t("Inspections per mile"),
                    description: t(
                      "Roadside inspections compared with the miles the carrier reports driving.",
                    ),
                    value: benchmarks.inspectionMileageAnomaly,
                  },
                  {
                    id: "units",
                    label: t("Inspected units"),
                    description: t(
                      "Distinct units seen at inspections compared with the power units on file.",
                    ),
                    value: benchmarks.inspectedUnitsAnomaly,
                  },
                  {
                    id: "mileage",
                    label: t("Miles per power unit"),
                    description: t("Reported mileage compared with peers of the same fleet size."),
                    value: benchmarks.powerUnitMileageAnomaly,
                  },
                ].map((row) => (
                  <li key={row.id} className="flex items-start gap-2.5 py-2">
                    <StatusDot tone={benchmarkTone(row.value)} className="mt-1.5" />
                    <div className="flex min-w-0 flex-1 flex-col gap-0.5">
                      <span className="text-sm">{row.label}</span>
                      <span className="text-muted-foreground text-xs">{row.description}</span>
                    </div>
                    <span className="text-muted-foreground shrink-0 text-xs">
                      {row.value === true
                        ? t("Anomaly")
                        : row.value === false
                          ? t("Within range")
                          : t("Not scored")}
                    </span>
                  </li>
                ))}
              </ul>
            ) : null,
        },
      ]}
    />
  );
}
