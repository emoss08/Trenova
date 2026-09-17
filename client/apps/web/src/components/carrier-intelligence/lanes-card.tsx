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
import { formatPercent } from "@trenova/shared/lib/utils";
import { RouteIcon } from "lucide-react";
import {
  IntelDate,
  IntelField,
  IntelFieldGrid,
  IntelNumber,
  IntelSectionCard,
} from "./intel-section-card";

export type LanesCardProps = {
  profile: CarrierIntelProfile;
  provider: string | null | undefined;
  className?: string;
};

function PercentValue({ value }: { value: number | null }) {
  if (value === null) {
    return <span className="text-muted-foreground">-</span>;
  }
  return <span className="tabular-nums">{formatPercent(value)}</span>;
}

export function LanesCard({ profile, provider, className }: LanesCardProps) {
  const t = useT();
  const lanes = profile.lanes;
  const preferred = lanes?.preferred ?? [];

  return (
    <IntelSectionCard
      title={t("Lanes")}
      icon={RouteIcon}
      coverage={profile.coverage}
      provider={provider}
      className={className}
      parts={[
        {
          section: "Lanes",
          hasData: lanes !== null,
          content: lanes ? (
            <div className="flex flex-col gap-3">
              <IntelFieldGrid className="sm:grid-cols-3">
                <IntelField label={t("Loads observed")}>
                  <IntelNumber value={lanes.totalLoads} />
                </IntelField>
                <IntelField label={t("First load")}>
                  <IntelDate value={lanes.firstLoadAt} />
                </IntelField>
                <IntelField label={t("Last load")}>
                  <IntelDate value={lanes.lastLoadAt} />
                </IntelField>
                <IntelField label={t("Full truckload")}>
                  <PercentValue value={lanes.ftlPercent} />
                </IntelField>
                <IntelField label={t("Less than truckload")}>
                  <PercentValue value={lanes.ltlPercent} />
                </IntelField>
                <IntelField label={t("Deadhead")}>
                  <PercentValue value={lanes.deadheadPercent} />
                </IntelField>
              </IntelFieldGrid>
              {preferred.length > 0 ? (
                <div className="flex flex-col gap-1">
                  <h4 className="text-muted-foreground text-xs font-medium">
                    {t("Preferred lanes")}
                  </h4>
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>{t("Origin")}</TableHead>
                        <TableHead>{t("Destination")}</TableHead>
                        <TableHead className="text-right">{t("Loads")}</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {preferred.map((lane, index) => (
                        <TableRow
                          key={`${lane.originCity ?? ""}-${lane.originState ?? ""}-${lane.destinationCity ?? ""}-${lane.destinationState ?? ""}-${index}`}
                        >
                          <TableCell>
                            {[lane.originCity, lane.originState].filter(Boolean).join(", ") || "-"}
                          </TableCell>
                          <TableCell>
                            {[lane.destinationCity, lane.destinationState]
                              .filter(Boolean)
                              .join(", ") || "-"}
                          </TableCell>
                          <TableCell className="text-right tabular-nums">
                            <IntelNumber value={lane.loads} />
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </div>
              ) : null}
            </div>
          ) : null,
        },
      ]}
    />
  );
}
