import { INTEL_EMPTY_VALUE, cityStateLabel } from "@/lib/carrier-intelligence";
import { useT } from "@trenova/shared/i18n/use-t";
import {
  Fact,
  FactGrid,
  MiniTable,
  SectionNote,
  SubHeading,
  dateOrDash,
  listOrDash,
  numberOrDash,
  percentOrDash,
} from "../intel-facts";
import { ProfileTab, type ProfileSectionProps } from "./profile-tab";

export function LanesSection({ profile, provider }: ProfileSectionProps) {
  const t = useT();
  const lanes = profile.lanes;

  return (
    <ProfileTab
      profile={profile}
      provider={provider}
      parts={[
        {
          section: "Lanes",
          hasData: lanes !== null,
          render: () => {
            if (!lanes) {
              return null;
            }
            const preferred = lanes.preferred ?? [];
            const preferredStates = lanes.preferredStates ?? [];
            const hasVolume = [
              lanes.totalLoads,
              lanes.firstLoadAt,
              lanes.lastLoadAt,
              lanes.ftlPercent,
              lanes.ltlPercent,
              lanes.deadheadPercent,
            ].some((value) => value !== null);
            const showLoads = preferred.some((lane) => lane.loads !== null);
            return (
              <div className="flex flex-col gap-5">
                {hasVolume ? (
                  <FactGrid className="grid-cols-2 sm:grid-cols-3">
                    <Fact label={t("Loads observed")}>{numberOrDash(lanes.totalLoads)}</Fact>
                    <Fact label={t("First load")}>{dateOrDash(lanes.firstLoadAt)}</Fact>
                    <Fact label={t("Last load")}>{dateOrDash(lanes.lastLoadAt)}</Fact>
                    <Fact label={t("Truckload")}>{percentOrDash(lanes.ftlPercent)}</Fact>
                    <Fact label={t("LTL")}>{percentOrDash(lanes.ltlPercent)}</Fact>
                    <Fact label={t("Deadhead")}>{percentOrDash(lanes.deadheadPercent)}</Fact>
                  </FactGrid>
                ) : null}
                <FactGrid className="sm:grid-cols-1">
                  <Fact label={t("Preferred states")} wrap>
                    {listOrDash(preferredStates)}
                  </Fact>
                </FactGrid>
                <div className="flex flex-col gap-1">
                  <SubHeading>{t("Preferred lanes")}</SubHeading>
                  {preferred.length > 0 ? (
                    <MiniTable
                      columns={[
                        { id: "lane", label: t("Lane") },
                        ...(showLoads
                          ? [{ id: "loads", label: t("Loads"), align: "right" as const }]
                          : []),
                      ]}
                      rows={preferred.map((lane, index) => ({
                        id: `${lane.originState ?? ""}-${lane.destinationState ?? ""}-${index}`,
                        cells: [
                          `${cityStateLabel(lane.originCity, lane.originState) ?? INTEL_EMPTY_VALUE} → ${
                            cityStateLabel(lane.destinationCity, lane.destinationState) ??
                            INTEL_EMPTY_VALUE
                          }`,
                          ...(showLoads ? [numberOrDash(lane.loads)] : []),
                        ],
                      }))}
                    />
                  ) : (
                    <SectionNote>{t("No preferred lanes on record.")}</SectionNote>
                  )}
                </div>
              </div>
            );
          },
        },
      ]}
    />
  );
}
