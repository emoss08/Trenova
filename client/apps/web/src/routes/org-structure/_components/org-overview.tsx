import { useT } from "@trenova/shared/i18n/use-t";
import { InfoPopover } from "@/components/info-popover";
import { KpiCard, KpiHeader, KpiSub } from "@/components/kpi/kpi-card";
import type {
  ApprovalDelegationRow,
  HeadcountSummary,
  JobPositionRow,
} from "@/lib/graphql/org-structure";
import { coverSummary, headcountSummary, largestGroup, vacantPositions } from "@/lib/org-chart";
import NumberFlow from "@number-flow/react";
import { CompositionBar } from "@trenova/shared/components/ui/composition-bar";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { BriefcaseIcon, Building2Icon, HandshakeIcon, UsersIcon } from "lucide-react";
import { useMemo } from "react";

const VALUE_CLASS = "font-mono text-[26px] leading-none font-semibold tracking-tight tabular-nums";

type OrgOverviewProps = {
  headcount: HeadcountSummary | undefined;
  positions: readonly JobPositionRow[] | undefined;
  /** Cover the signed-in user is party to, both directions; absent when they may not read it. */
  delegations: readonly ApprovalDelegationRow[] | undefined;
  showCover: boolean;
  now: number;
};

/**
 * The organisation in four numbers: how many people, how the work is titled,
 * where they sit, and who is approving in whose place. Each is read from the
 * same lists the panels below draw, so the strip never disagrees with them.
 */
export function OrgOverview({
  headcount,
  positions,
  delegations,
  showCover,
  now,
}: OrgOverviewProps) {
  const t = useT();

  const people = useMemo(
    () => headcountSummary(headcount ?? { activeTotal: 0, driverTotal: 0, terminated: 0 }),
    [headcount],
  );
  const activePositions = (positions ?? []).filter(
    (position) => position.status === "Active",
  ).length;
  const archivedPositions = (positions ?? []).length - activePositions;
  const vacant = useMemo(
    () => vacantPositions(positions ?? [], headcount?.byPosition ?? []).length,
    [positions, headcount],
  );
  const largestTerminal = useMemo(() => largestGroup(headcount?.byFleet ?? []), [headcount]);
  const cover = useMemo(() => coverSummary(delegations ?? [], now), [delegations, now]);

  return (
    <div
      className={
        showCover
          ? "grid grid-cols-4 gap-3 lg:grid-cols-8"
          : "grid grid-cols-4 gap-3 lg:grid-cols-6"
      }
    >
      <KpiCard span={2}>
        <KpiHeader
          icon={<UsersIcon className="size-[11px]" />}
          label={t("People")}
          info={
            <InfoPopover title={t("People")}>
              {t("Active workers plus users holding a position. Terminated workers are left out.")}
            </InfoPopover>
          }
        />
        {headcount ? (
          <NumberFlow value={people.people} className={VALUE_CLASS} aria-label={t("People")} />
        ) : (
          <Skeleton className="h-6.5 w-10" />
        )}
        <CompositionBar
          size="sm"
          className="mt-auto"
          aria-label={t("Who they are")}
          segments={[
            { key: "drivers", label: t("Drivers"), value: people.drivers },
            { key: "other", label: t("Other workers"), value: people.nonDriving },
            { key: "staff", label: t("Staff"), value: people.staff },
          ]}
        />
      </KpiCard>

      <KpiCard span={2}>
        <KpiHeader
          icon={<BriefcaseIcon className="size-[11px]" />}
          label={t("Positions")}
          info={
            <InfoPopover title={t("Positions")}>
              {t(
                "Positions still open, driving and non-driving, whether or not somebody holds them. Retired titles are not counted.",
              )}
            </InfoPopover>
          }
        />
        {positions ? (
          <NumberFlow value={activePositions} className={VALUE_CLASS} aria-label={t("Positions")} />
        ) : (
          <Skeleton className="h-6.5 w-10" />
        )}
        <KpiSub>
          {describePositions(vacant, archivedPositions, Boolean(positions && headcount))}
        </KpiSub>
      </KpiCard>

      <KpiCard span={2}>
        <KpiHeader
          icon={<Building2Icon className="size-[11px]" />}
          label={t("Terminals")}
          info={
            <InfoPopover title={t("Terminals")}>
              {t("Terminals with at least one active worker assigned.")}
            </InfoPopover>
          }
        />
        {headcount ? (
          <NumberFlow
            value={headcount.byFleet.length}
            className={VALUE_CLASS}
            aria-label={t("Terminals")}
          />
        ) : (
          <Skeleton className="h-6.5 w-10" />
        )}
        <KpiSub>
          {largestTerminal
            ? t("Largest: {0}, {1} people", largestTerminal.label, largestTerminal.workers)
            : headcount && people.terminated > 0
              ? t("{0} off the roster", people.terminated)
              : t("Nobody on the roster yet")}
        </KpiSub>
      </KpiCard>

      {showCover ? (
        <KpiCard span={2}>
          <KpiHeader
            icon={<HandshakeIcon className="size-[11px]" />}
            label={t("Cover in force")}
            info={
              <InfoPopover title={t("Cover in force")}>
                {t(
                  "Delegations running today. Ones scheduled to start later and ones that have ended do not count.",
                )}
              </InfoPopover>
            }
          />
          {delegations ? (
            <NumberFlow
              value={cover.active}
              className={VALUE_CLASS}
              aria-label={t("Cover in force")}
            />
          ) : (
            <Skeleton className="h-6.5 w-10" />
          )}
          <KpiSub>{describeCover(cover)}</KpiSub>
        </KpiCard>
      ) : null}
    </div>
  );
}

function describePositions(vacant: number, archived: number, loaded: boolean): string {
  if (!loaded) return "Titles the roster is counted by";
  const parts: string[] = [];
  if (vacant > 0) parts.push(`${vacant} ${vacant === 1 ? "title" : "titles"} nobody holds`);
  if (archived > 0) parts.push(`${archived} archived`);
  return parts.length > 0 ? parts.join(" · ") : "Every open position is filled";
}

function describeCover(cover: ReturnType<typeof coverSummary>): string {
  const parts: string[] = [];
  if (cover.endingSoon > 0) parts.push(`${cover.endingSoon} ending this week`);
  if (cover.openEnded > 0) parts.push(`${cover.openEnded} until called back`);
  if (cover.scheduled > 0) parts.push(`${cover.scheduled} starting later`);
  return parts.length > 0 ? parts.join(" · ") : "Nobody is approving in anybody's place";
}
