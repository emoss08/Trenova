import { useT } from "@trenova/shared/i18n/use-t";
import type { CarrierIntelProfile } from "@/lib/graphql/carrier-intelligence";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@trenova/shared/components/ui/table";
import { cn, formatPercent } from "@trenova/shared/lib/utils";
import { ClipboardCheckIcon } from "lucide-react";
import {
  IntelDate,
  IntelField,
  IntelFieldGrid,
  IntelNumber,
  IntelSectionCard,
} from "./intel-section-card";

export type InspectionsCrashesCardProps = {
  profile: CarrierIntelProfile;
  provider: string | null | undefined;
  className?: string;
};

type OosRowProps = {
  label: string;
  inspections: number | null;
  outOfService: number | null;
  rate: number | null;
  national: number | null;
};

function OosRow({ label, inspections, outOfService, rate, national }: OosRowProps) {
  const t = useT();
  const aboveNational = rate !== null && national !== null && rate > national;

  return (
    <TableRow>
      <TableCell className="font-medium">{label}</TableCell>
      <TableCell className="text-right tabular-nums">
        <IntelNumber value={inspections} />
      </TableCell>
      <TableCell className="text-right tabular-nums">
        <IntelNumber value={outOfService} />
      </TableCell>
      <TableCell
        className={cn(
          "text-right tabular-nums",
          aboveNational && "font-medium text-red-700 dark:text-red-400",
        )}
        title={aboveNational ? t("Above the national average") : undefined}
      >
        {rate !== null ? formatPercent(rate) : "-"}
      </TableCell>
      <TableCell className="text-muted-foreground text-right tabular-nums">
        {national !== null ? formatPercent(national) : "-"}
      </TableCell>
    </TableRow>
  );
}

export function InspectionsCrashesCard({
  profile,
  provider,
  className,
}: InspectionsCrashesCardProps) {
  const t = useT();
  const inspections = profile.inspections;
  const crashes = profile.crashes;
  const fatal = (crashes?.fatal ?? 0) > 0;

  return (
    <IntelSectionCard
      title={t("Inspections & crashes")}
      icon={ClipboardCheckIcon}
      coverage={profile.coverage}
      provider={provider}
      emphasis={fatal ? "warning" : "none"}
      className={className}
      parts={[
        {
          section: "Inspections",
          heading: t("Inspections"),
          hasData: inspections !== null,
          content: inspections ? (
            <div className="flex flex-col gap-2">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>{t("Type")}</TableHead>
                    <TableHead className="text-right">{t("Inspections")}</TableHead>
                    <TableHead className="text-right">{t("OOS")}</TableHead>
                    <TableHead className="text-right">{t("OOS rate")}</TableHead>
                    <TableHead className="text-right">{t("National")}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  <OosRow
                    label={t("Driver")}
                    inspections={inspections.driver}
                    outOfService={inspections.driverOos}
                    rate={inspections.driverOosRate}
                    national={inspections.nationalDriverOosRate}
                  />
                  <OosRow
                    label={t("Vehicle")}
                    inspections={inspections.vehicle}
                    outOfService={inspections.vehicleOos}
                    rate={inspections.vehicleOosRate}
                    national={inspections.nationalVehicleOosRate}
                  />
                  <OosRow
                    label={t("Hazmat")}
                    inspections={inspections.hazmat}
                    outOfService={inspections.hazmatOos}
                    rate={inspections.hazmatOosRate}
                    national={inspections.nationalHazmatOosRate}
                  />
                </TableBody>
              </Table>
              <IntelFieldGrid>
                <IntelField label={t("Total inspections")}>
                  <IntelNumber value={inspections.total} />
                </IntelField>
                <IntelField label={t("Last inspection")}>
                  <IntelDate value={inspections.lastInspectionAt} />
                </IntelField>
              </IntelFieldGrid>
            </div>
          ) : null,
        },
        {
          section: "Crashes",
          heading: t("Crashes"),
          hasData: crashes !== null,
          content: crashes ? (
            <IntelFieldGrid className="sm:grid-cols-3">
              <IntelField label={t("Total")}>
                <IntelNumber value={crashes.total} />
              </IntelField>
              <IntelField label={t("Fatal")}>
                <span className={cn(fatal && "font-medium text-red-700 dark:text-red-400")}>
                  <IntelNumber value={crashes.fatal} />
                </span>
              </IntelField>
              <IntelField label={t("Injury")}>
                <IntelNumber value={crashes.injury} />
              </IntelField>
              <IntelField label={t("Tow-away")}>
                <IntelNumber value={crashes.tow} />
              </IntelField>
              <IntelField label={t("Last crash")}>
                <IntelDate value={crashes.lastCrashAt} />
              </IntelField>
            </IntelFieldGrid>
          ) : null,
        },
      ]}
    />
  );
}
