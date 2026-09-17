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
import { TruckIcon } from "lucide-react";
import { IntelField, IntelFieldGrid, IntelNumber, IntelSectionCard } from "./intel-section-card";

export type FleetEquipmentCardProps = {
  profile: CarrierIntelProfile;
  provider: string | null | undefined;
  className?: string;
};

export function FleetEquipmentCard({ profile, provider, className }: FleetEquipmentCardProps) {
  const t = useT();
  const fleet = profile.fleet;
  const equipment = profile.equipment ?? [];

  return (
    <IntelSectionCard
      title={t("Fleet & equipment")}
      icon={TruckIcon}
      coverage={profile.coverage}
      provider={provider}
      className={className}
      parts={[
        {
          section: "Fleet",
          heading: t("Fleet"),
          hasData: fleet !== null,
          content: fleet ? (
            <IntelFieldGrid className="sm:grid-cols-3">
              <IntelField label={t("Power units")}>
                <IntelNumber value={fleet.powerUnits} />
              </IntelField>
              <IntelField label={t("Drivers")}>
                <IntelNumber value={fleet.drivers} />
              </IntelField>
              <IntelField label={t("CDL drivers")}>
                <IntelNumber value={fleet.cdlDrivers} />
              </IntelField>
              <IntelField label={t("Trucks")}>
                <IntelNumber value={fleet.trucks} />
              </IntelField>
              <IntelField label={t("Trailers")}>
                <IntelNumber value={fleet.trailers} />
              </IntelField>
              <IntelField label={t("Owned tractors")}>
                <IntelNumber value={fleet.ownedTractors} />
              </IntelField>
              <IntelField label={t("Term-leased tractors")}>
                <IntelNumber value={fleet.termLeasedTractors} />
              </IntelField>
              <IntelField label={t("Owned trailers")}>
                <IntelNumber value={fleet.ownedTrailers} />
              </IntelField>
              <IntelField label={t("Term-leased trailers")}>
                <IntelNumber value={fleet.termLeasedTrailers} />
              </IntelField>
            </IntelFieldGrid>
          ) : null,
        },
        {
          section: "Equipment",
          heading: t("Registered equipment"),
          hasData: equipment.length > 0,
          content: (
            <div className="max-h-72 overflow-auto rounded-md border">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>{t("Unit")}</TableHead>
                    <TableHead>{t("Type")}</TableHead>
                    <TableHead>{t("Make / model")}</TableHead>
                    <TableHead className="text-right">{t("Year")}</TableHead>
                    <TableHead>{t("VIN")}</TableHead>
                    <TableHead>{t("Plate")}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {equipment.map((unit, index) => (
                    <TableRow key={unit.vin ?? `${unit.unitNumber ?? "unit"}-${index}`}>
                      <TableCell>{unit.unitNumber ?? "-"}</TableCell>
                      <TableCell>
                        {[unit.unitType, unit.category].filter(Boolean).join(" · ") || "-"}
                      </TableCell>
                      <TableCell>
                        {[unit.make, unit.model].filter(Boolean).join(" ") || "-"}
                      </TableCell>
                      <TableCell className="text-right tabular-nums">{unit.year ?? "-"}</TableCell>
                      <TableCell className="font-mono text-xs">{unit.vin ?? "-"}</TableCell>
                      <TableCell>
                        {unit.plateNumber
                          ? `${unit.plateNumber}${unit.plateState ? ` (${unit.plateState})` : ""}`
                          : "-"}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          ),
        },
      ]}
    />
  );
}
