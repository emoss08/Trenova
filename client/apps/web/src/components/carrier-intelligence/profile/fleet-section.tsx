import { joinPresent } from "@/lib/carrier-intelligence";
import { useT } from "@trenova/shared/i18n/use-t";
import { Fact, FactGrid, MiniTable, numberOrDash, textOrDash } from "../intel-facts";
import { ProfileTab, type ProfileSectionProps } from "./profile-tab";

export function FleetSection({ profile, provider }: ProfileSectionProps) {
  const t = useT();
  const fleet = profile.fleet;
  const equipment = profile.equipment ?? [];

  return (
    <ProfileTab
      profile={profile}
      provider={provider}
      parts={[
        {
          section: "Fleet",
          title: t("Fleet"),
          hasData: fleet !== null,
          render: () =>
            fleet ? (
              <FactGrid className="grid-cols-2 sm:grid-cols-3">
                <Fact label={t("Power units")}>{numberOrDash(fleet.powerUnits)}</Fact>
                <Fact label={t("Drivers")}>{numberOrDash(fleet.drivers)}</Fact>
                <Fact label={t("CDL drivers")}>{numberOrDash(fleet.cdlDrivers)}</Fact>
                <Fact label={t("Trucks")}>{numberOrDash(fleet.trucks)}</Fact>
                <Fact label={t("Trailers")}>{numberOrDash(fleet.trailers)}</Fact>
                <Fact label={t("Owned tractors")}>{numberOrDash(fleet.ownedTractors)}</Fact>
                <Fact label={t("Term-leased tractors")}>
                  {numberOrDash(fleet.termLeasedTractors)}
                </Fact>
                <Fact label={t("Owned trailers")}>{numberOrDash(fleet.ownedTrailers)}</Fact>
                <Fact label={t("Term-leased trailers")}>
                  {numberOrDash(fleet.termLeasedTrailers)}
                </Fact>
              </FactGrid>
            ) : null,
        },
        {
          section: "Equipment",
          title: t("Registered equipment"),
          hasData: equipment.length > 0,
          render: () => (
            <div className="max-h-80 overflow-y-auto">
              <MiniTable
                columns={[
                  { id: "unit", label: t("Unit") },
                  { id: "type", label: t("Type") },
                  { id: "vehicle", label: t("Vehicle") },
                  { id: "vin", label: t("VIN") },
                  { id: "plate", label: t("Plate") },
                ]}
                rows={equipment.map((unit, index) => ({
                  id: `${unit.vin ?? unit.unitNumber ?? "unit"}-${index}`,
                  cells: [
                    textOrDash(unit.unitNumber),
                    textOrDash(joinPresent([unit.unitType, unit.category], " · ")),
                    textOrDash(joinPresent([unit.year, unit.make, unit.model], " ")),
                    <span key="vin" className="font-mono text-xs">
                      {textOrDash(unit.vin)}
                    </span>,
                    textOrDash(
                      unit.plateNumber
                        ? joinPresent(
                            [unit.plateNumber, unit.plateState ? `(${unit.plateState})` : null],
                            " ",
                          )
                        : null,
                    ),
                  ],
                }))}
              />
            </div>
          ),
        },
      ]}
    />
  );
}
