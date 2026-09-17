import { useT } from "@trenova/shared/i18n/use-t";
import type { CarrierIntelProfile } from "@/lib/graphql/carrier-intelligence";
import { cn } from "@trenova/shared/lib/utils";
import { CircleCheckIcon, GaugeIcon, TriangleAlertIcon } from "lucide-react";
import { IntelSectionCard } from "./intel-section-card";

export type BenchmarksCardProps = {
  profile: CarrierIntelProfile;
  provider: string | null | undefined;
  className?: string;
};

function AnomalyRow({
  label,
  description,
  value,
}: {
  label: string;
  description: string;
  value: boolean | null;
}) {
  const t = useT();

  return (
    <li className="flex items-start gap-2 px-2.5 py-2">
      {value === true ? (
        <TriangleAlertIcon className="mt-0.5 size-4 shrink-0 text-yellow-600" aria-hidden />
      ) : value === false ? (
        <CircleCheckIcon className="mt-0.5 size-4 shrink-0 text-green-600" aria-hidden />
      ) : (
        <span className="text-muted-foreground mt-0.5 size-4 shrink-0 text-center">-</span>
      )}
      <span className="flex min-w-0 flex-col">
        <span className={cn("text-sm", value === true && "font-medium")}>
          {label}
          <span className="text-muted-foreground">
            {" · "}
            {value === true ? t("Anomaly") : value === false ? t("Within range") : t("Not scored")}
          </span>
        </span>
        <span className="text-muted-foreground text-xs">{description}</span>
      </span>
    </li>
  );
}

export function BenchmarksCard({ profile, provider, className }: BenchmarksCardProps) {
  const t = useT();
  const benchmarks = profile.benchmarks;

  return (
    <IntelSectionCard
      title={t("Benchmarks")}
      icon={GaugeIcon}
      coverage={profile.coverage}
      provider={provider}
      emphasis={benchmarks?.anyAnomaly ? "warning" : "none"}
      className={className}
      parts={[
        {
          section: "Benchmarks",
          hasData: benchmarks !== null,
          content: benchmarks ? (
            <ul className="divide-y rounded-md border">
              <AnomalyRow
                label={t("Inspections per mile")}
                description={t(
                  "Roadside inspections compared with the miles the carrier reports driving.",
                )}
                value={benchmarks.inspectionMileageAnomaly}
              />
              <AnomalyRow
                label={t("Inspected units")}
                description={t(
                  "Distinct units seen at inspections compared with the power units on file.",
                )}
                value={benchmarks.inspectedUnitsAnomaly}
              />
              <AnomalyRow
                label={t("Miles per power unit")}
                description={t("Reported mileage compared with peers of the same fleet size.")}
                value={benchmarks.powerUnitMileageAnomaly}
              />
            </ul>
          ) : null,
        },
      ]}
    />
  );
}
